## Moteur de calcul Bilan Carbone avancé (design & snippets)

**Ce projet s’inspire du dépôt GitHub suivant : `https://github.com/incubateur-ademe/nosgestesclimat`. Les modules, conventions et certains scripts d’ETL ont été adaptés de ce dépôt comme point de départ — toute modification doit être tracée dans l’historique Git.**  
Si l’URL doit être injectée dynamiquement, utiliser le placeholder `{{github_repo_url}}` et le remplacer par `https://github.com/incubateur-ademe/nosgestesclimat` par défaut.

Les paramètres suivants doivent être acceptés partout (API, CLI, notebooks, README) :

- `github_repo_url: string`
- `tenant_name: string`
- `primary_fe_namespace: "ADEME" | "DEFRA" | "Ecoinvent" | "custom"`
- `default_gwp: "AR5" | "AR6"`
- `compute_mode: "local" | "cluster" | "serverless"`

---

### 1. Spécification fonctionnelle & architecture

Objectif : moteur de calcul GES **multi-tenant**, **traçable**, **reproductible**, avec **incertitude** documentée.

- **Données d’activité**: flux bruts (compta, énergie, achats, déplacements…) stockés en Parquet/Delta par tenant.
- **Facteurs d’émission (FE)**: table normalisée (Postgres/Delta) avec namespace, valeur, unité, incertitude, période de validité.
- **Moteur de calcul**:
  - mode déterministe: `emission = activity * FE.value`
  - mode Monte-Carlo: `FE ~ distribution` → distributions par poste, scope et période.
  - sensibilité (Sobol) + option Bayésienne (PyMC/Stan) pour calibrer certains FE.
- **Infra data**:
  - Data Lake Parquet/Delta sur S3/GCS.
  - Spark/Delta **ou** Ray + Polars/DuckDB selon `compute_mode`.
  - Arrow comme format d’échange entre processus (Python ↔ Rust).
- **Provenance & versionning**:
  - chaque calcul garde un **snapshot de FE**, un **hash SHA256** des inputs + code + paramètres.
  - JSON `provenance.json` conforme W3C PROV.
- **API v1**:
  - `/api/v1/tenants/{tenant}/datasets`
  - `/api/v1/tenants/{tenant}/calculations`
  - `/api/v1/tenants/{tenant}/reports`

Les résultats agrégés (par tenant / année / scope / poste) sont exposés à ton UI Next.js actuelle via `/app/bilan`, `/app/strategy`, `/app/acv`, `/app/esg`.

---

### 2. Pseudo-code exécutable & optimisé (vue haut niveau)

#### 2.1 Ingestion dataset

```python
def ingest_dataset(path: str, tenant_name: str, github_repo_url: str,
                   compute_profile: str = "fast") -> DatasetMeta:
    table = read_csv_or_parquet(path)              # Polars / Spark
    table = normalize_schema(table)                # colonnes standardisées
    validate_schema(table)                         # types, ranges, dates
    dataset_id = uuid4()
    write_delta(f"s3://datalake/{tenant_name}/datasets/{dataset_id}", table)

    return DatasetMeta(
        id=dataset_id,
        tenant=tenant_name,
        n_rows=table.height,
        github_repo_url=github_repo_url,
        compute_profile=compute_profile,
    )
```

#### 2.2 Calcul déterministe + Monte-Carlo

```python
def run_calculation(dataset: ActivityTable,
                    fe_table: FactorTable,
                    mode: Literal["deterministic", "montecarlo"],
                    mc_samples: int = 10_000,
                    default_gwp: str = "AR6") -> CalculationResult:
    # 1) jointure activity ↔ FE
    joined = dataset.join(fe_table, on=["fe_key"], how="left")

    if mode == "deterministic":
        joined["em_kg"] = joined["quantity"] * joined["fe_value_kg_per_unit"]
        return aggregate_results(joined)

    # 2) Monte-Carlo (Ray ou Spark)
    def mc_worker(seed: int) -> PartialResult:
        np.random.seed(seed)
        fe_mu = joined["fe_value_kg_per_unit"].to_numpy()
        fe_sigma = fe_mu * joined["uncertainty_pct"].to_numpy()

        em_samples = []
        for _ in range(mc_samples_per_worker(mc_samples)):
            fe_sample = np.random.normal(fe_mu, fe_sigma)
            em = joined["quantity"].to_numpy() * fe_sample
            em_samples.append(em)
        return aggregate_mc(np.stack(em_samples, axis=0), joined)

    partials = ray.get([
        mc_worker.remote(seed) for seed in seeds_for(mc_samples)
    ])
    return merge_mc_partials(partials)
```

#### 2.3 Signature de reproductibilité

```python
def build_signature(raw_inputs_hash: str,
                    fe_snapshot_hash: str,
                    git_commit: str,
                    compute_params: dict) -> str:
    payload = json.dumps({
        "inputs": raw_inputs_hash,
        "fe": fe_snapshot_hash,
        "git": git_commit,
        "params": compute_params,
    }, sort_keys=True).encode("utf-8")
    return "sha256-" + hashlib.sha256(payload).hexdigest()
```

---

### 3. Snippets de code prêts à intégrer

#### 3.1 Modèle FE (Postgres / SQL)

```sql
CREATE TABLE IF NOT EXISTS emission_factors (
  id              UUID PRIMARY KEY,
  namespace       TEXT NOT NULL, -- 'ADEME', 'DEFRA', 'Ecoinvent', 'custom'
  name            TEXT NOT NULL,
  value           NUMERIC(18,6) NOT NULL, -- kgCO2e / unit
  unit            TEXT NOT NULL,
  scope_applicable SMALLINT[] NOT NULL,
  source_url      TEXT,
  valid_from      DATE NOT NULL,
  valid_to        DATE NOT NULL,
  uncertainty_pct NUMERIC(6,4) NOT NULL DEFAULT 0,
  metadata        JSONB,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

#### 3.2 Python – calcul simple + Monte-Carlo local

```python
import polars as pl
import numpy as np

def deterministic_emissions(activities: pl.DataFrame,
                            factors: pl.DataFrame) -> pl.DataFrame:
    joined = activities.join(factors, on="fe_id", how="left")
    return (
        joined
        .with_columns(
            (pl.col("quantity") * pl.col("value")).alias("em_kgCO2e")
        )
        .groupby(["tenant_id", "year", "scope", "category"])
        .agg(pl.col("em_kgCO2e").sum().alias("total_kgCO2e"))
    )

def monte_carlo_emissions(joined: pl.DataFrame,
                          samples: int = 10_000) -> dict:
    qty = joined["quantity"].to_numpy()
    mu = joined["value"].to_numpy()
    sigma = mu * joined["uncertainty_pct"].to_numpy()

    draws = np.random.normal(mu, sigma, size=(samples, len(mu)))
    em_samples = draws * qty

    total = em_samples.sum(axis=1)  # total par tirage
    return {
        "mean": float(total.mean()),
        "median": float(np.median(total)),
        "ci_90": [float(np.percentile(total, 5)),
                  float(np.percentile(total, 95))],
    }
```

#### 3.3 Rust – noyau de multiplication vectorielle (via PyO3)

```rust
use pyo3::prelude::*;

#[pyfunction]
fn mul_activity_fe(quantity: Vec<f64>, fe_value: Vec<f64>) -> PyResult<Vec<f64>> {
    if quantity.len() != fe_value.len() {
        return Err(pyo3::exceptions::PyValueError::new_err("length mismatch"));
    }
    Ok(quantity.iter().zip(fe_value.iter())
       .map(|(q, f)| q * f)
       .collect())
}

#[pymodule]
fn carbon_kernels(_py: Python, m: &PyModule) -> PyResult<()> {
    m.add_function(wrap_pyfunction!(mul_activity_fe, m)?)?;
    Ok(())
}
```

#### 3.4 JS (Next.js) – affichage d’une distribution Monte-Carlo

```ts
type McSummary = {
  total_mean_kgCO2e: number;
  ci_90: [number, number];
};

export function McBadge({ summary }: { summary: McSummary }) {
  return (
    <div className="rounded-full border border-emerald-500/50 bg-emerald-500/10 px-3 py-1 text-xs text-emerald-200">
      {summary.total_mean_kgCO2e.toFixed(0)} kgCO₂e
      {" · "}
      CI₉₀ [{summary.ci_90[0].toFixed(0)} – {summary.ci_90[1].toFixed(0)}]
    </div>
  );
}
```

---

### 4. JSON schemas request / response

#### 4.1 `POST /api/v1/tenants/{tenant}/datasets`

**Request**:

```json
{
  "source": "file",
  "path": "s3://bucket/path/to/file.parquet",
  "github_repo_url": "{{github_repo_url}}",
  "compute_profile": "fast"
}
```

**Response**:

```json
{
  "dataset_id": "ds-123",
  "tenant": "acme-saas",
  "n_rows": 123456,
  "ingest_job_id": "job-abc",
  "compute_profile": "fast"
}
```

#### 4.2 `POST /api/v1/tenants/{tenant}/calculations`

**Request**:

```json
{
  "dataset_id": "ds-123",
  "period": "2024",
  "mode": "async",
  "compute_profile": "montecarlo",
  "montecarlo": { "samples": 10000, "seed": 42 },
  "gwp": "AR6",
  "primary_fe_namespace": "ADEME",
  "fe_namespace_priority": ["ADEME", "custom"],
  "repro_signature_note": "inspired_by_github: {{github_repo_url}}"
}
```

**Response**:

```json
{
  "calculation_id": "calc-uuid",
  "calculation_signature": "sha256-...",
  "tenant_id": "acme-saas",
  "estimated_work_units": 12.5,
  "status": "queued"
}
```

#### 4.3 Résultat détaillé (GET `/api/v1/calculations/{id}`)

```json
{
  "calculation_id": "calc-uuid",
  "signature": "sha256-...",
  "tenant_id": "acme-saas",
  "period": "2024",
  "summary": {
    "total_mean_kgCO2e": 12345.6,
    "total_median_kgCO2e": 12200,
    "ci_90": [11000, 13500]
  },
  "breakdown": [
    {
      "category": "energy",
      "scope": 2,
      "mean": 5000,
      "ci_90": [4500, 5600],
      "n_lines": 123,
      "fe_used": { "id": "fe-uuid", "value": 0.052 }
    }
  ],
  "provenance": {
    "raw_inputs_hash": "sha256-...",
    "fe_snapshot_id": "fe-snap-20241207",
    "git_commit": "abcd1234",
    "docker_image": "saa-solver:v1.2.3",
    "github_repo_url": "{{github_repo_url}}"
  },
  "mc_samples_location": "s3://.../calc-uuid/samples.parquet"
}
```

---

### 5. Checklist d’audit et tests

**Unit tests (Python / Rust)**

- conversions : kWh → kgCO₂e, L carburant → kgCO₂e, tkm, m², etc.
- jointure activités ↔ FE (clé absente, namespace différent, date hors plage).
- Monte-Carlo : vérifier que l’écart type et CI90 se comportent comme attendu pour un FE synthétique.
- Rust kernels : multiplication vecteur-vecteur, agrégation scope/poste.

**Tests d’intégration**

- flux complet `ingest_dataset → run_calculation → signature → stockage`.
- re-jeu (`replay_script.sh --signature <sig>`) : les résultats doivent être strictement identiques.

**Tests de régression**

- comparer total annuel vs baseline (année précédente) avec tolérance (ex ±1%).

**Fuzz tests**

- données extrêmes (0, négatif, très grand), dates hors bornes, FE manquants.

---

### 6. SLO/SLI & recommandations d’optimisation

- **SLO API sync** (`mode=sync`, ≤ 10k lignes) :
  - p95 < 200 ms, p99 < 500 ms.
- **SLO jobs batch** (`mode=async`, ≥ 1M lignes) :
  - temps total < 30 min pour N=10k tirages Monte-Carlo sur cluster `compute_mode=cluster`.
- **SLI principaux** :
  - `job_duration_seconds{tenant, compute_profile}`
  - `rows_processed_total`
  - `mc_samples_per_second`
  - `calculation_errors_total`

**Optimisation** :

- utiliser **Arrow** pour sérialiser les tables entre Python ↔ Rust ↔ Spark.
- partitionner Delta par `tenant_id`, `year`, `scope` pour minimiser l’I/O.
- pré-calculer des **materialized views** (par tenant/année/scope) et mettre en cache (Redis/RocksDB) pour les vues Next.js.

---

### 7. Intégration avec le site web actuel

- le backend Go actuel reste l’API **temps réel** de ton SaaS (auth, entrées, résumés simples).
- le moteur avancé décrit ici tourne comme un **service de calcul séparé** (Python/Rust/Spark), exposant `/api/v1/...`.
- le frontend Next.js peut appeler ce service pour :
  - les vues détaillées `/app/bilan` (résultats déterministes + MC),
  - `/app/strategy` (trajectoires issues des distributions MC),
  - `/app/acv` (analyses produit),
  - `/app/esg` (rapports CSRD/VSME/Ecovadis).

Les pages sont déjà en place ; il suffit ensuite de brancher les appels HTTP sur ces nouveaux endpoints lorsque tu seras prêt à déployer l’infra Spark/Ray décrite ci‑dessus.


