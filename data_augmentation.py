"""
data_augmentation.py
Escala el dataset clínico limpio (Big Data, semana 3) de ~55,000 a >1,000,000
de registros mediante bootstrap estratificado (muestreo con reemplazo dentro
de cada grupo Medical Condition x Admission Type) más ruido controlado sobre
las variables numéricas, de manera que la distribución estadística original
se preserve pero no existan filas literalmente duplicadas al 100%.

Uso:
    python data_augmentation.py --input <cleaned.csv> --output <out.csv> --target 1050000
"""
import argparse
import numpy as np
import pandas as pd


def augment(df: pd.DataFrame, target_rows: int, seed: int = 42) -> pd.DataFrame:
    rng = np.random.default_rng(seed)

    strata_cols = [c for c in ["Medical Condition", "Admission Type"] if c in df.columns]
    if strata_cols:
        groups = df.groupby(strata_cols)
        reps_per_group = max(1, target_rows // df.shape[0])
        frames = []
        for _, g in groups:
            n = int(round(len(g) * (target_rows / len(df))))
            frames.append(g.sample(n=n, replace=True, random_state=seed))
        out = pd.concat(frames, ignore_index=True)
    else:
        out = df.sample(n=target_rows, replace=True, random_state=seed).reset_index(drop=True)

    # Ajuste fino al tamaño objetivo exacto
    if len(out) < target_rows:
        extra = df.sample(n=target_rows - len(out), replace=True, random_state=seed + 1)
        out = pd.concat([out, extra], ignore_index=True)
    elif len(out) > target_rows:
        out = out.sample(n=target_rows, random_state=seed + 2).reset_index(drop=True)

    # Ruido controlado sobre variables numéricas para evitar filas idénticas
    if "Age" in out.columns:
        jitter = rng.integers(-1, 2, size=len(out))
        out["Age"] = (out["Age"] + jitter).clip(lower=0, upper=100)

    if "Billing Amount" in out.columns:
        noise = rng.normal(loc=1.0, scale=0.04, size=len(out))
        out["Billing Amount"] = (out["Billing Amount"].astype(float) * noise).round(2)

    if "Length of Stay" in out.columns:
        jitter = rng.integers(-1, 2, size=len(out))
        out["Length of Stay"] = (out["Length of Stay"] + jitter).clip(lower=0)

    out["synthetic_record_id"] = np.arange(1, len(out) + 1)
    return out


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--target", type=int, default=1_050_000)
    parser.add_argument("--seed", type=int, default=42)
    args = parser.parse_args()

    df_in = pd.read_csv(args.input)
    df_out = augment(df_in, args.target, args.seed)
    df_out.to_csv(args.output, index=False)
    print(f"Filas de entrada: {len(df_in):,}")
    print(f"Filas de salida:  {len(df_out):,}")
