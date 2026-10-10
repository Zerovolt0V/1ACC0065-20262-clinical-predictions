"""
data_cleaning.py
Reproduce, a partir del dataset original de Kaggle (Healthcare Dataset, 55 500
registros), la limpieza realizada en el curso de Big Data (semana 3) y añade el
tratamiento de los montos facturados negativos, produciendo el dataset limpio
que sirve de entrada a data_augmentation.py.

Pasos:
    1. Eliminar filas duplicadas exactas.
    2. Convertir las fechas de admisión y de alta a tipo fecha.
    3. Calcular 'Length of Stay' (días de estancia), truncando a cero los negativos.
    4. Derivar 'Admission Day of Week', 'Admission Month' y 'Admission Year'.
    5. Descartar columnas identificatorias o sin valor predictivo
       (Name, Doctor, Hospital, Date of Admission, Discharge Date, Room Number).
    6. Codificar cada columna de texto con enteros en orden alfabético.
    7. Eliminar registros con 'Billing Amount' negativo (artefacto del generador
       sintético del dataset original; no existe un valor real que recuperar).

Uso:
    python data_cleaning.py --input <healthcare_dataset.csv> --output <clean.csv> [--codebook <codebook.json>]
"""
import argparse
import json
import pandas as pd

DROP_COLS = ["Name", "Doctor", "Hospital", "Date of Admission", "Discharge Date", "Room Number"]
COLUMN_ORDER = [
    "Age", "Gender", "Blood Type", "Medical Condition", "Insurance Provider",
    "Billing Amount", "Admission Type", "Medication", "Test Results",
    "Length of Stay", "Admission Day of Week", "Admission Month", "Admission Year",
]


def clean(df: pd.DataFrame):
    stats = {"entrada": len(df)}

    # 1. Duplicados exactos
    df = df.drop_duplicates().copy()
    stats["sin_duplicados"] = len(df)

    # 2. Fechas
    df["Date of Admission"] = pd.to_datetime(df["Date of Admission"])
    df["Discharge Date"] = pd.to_datetime(df["Discharge Date"])

    # 3-4. Variables derivadas de la fecha de admisión
    df["Length of Stay"] = (df["Discharge Date"] - df["Date of Admission"]).dt.days.clip(lower=0)
    df["Admission Day of Week"] = df["Date of Admission"].dt.day_name()
    df["Admission Month"] = df["Date of Admission"].dt.month_name()
    df["Admission Year"] = df["Date of Admission"].dt.year

    # 5. Columnas identificatorias / sin valor predictivo
    df = df.drop(columns=DROP_COLS)

    # 6. Codificación entera alfabética (misma que el curso de Big Data); se hace
    #    antes de filtrar para que los códigos no dependan de las filas eliminadas.
    codebook = {}
    for col in df.select_dtypes(include="object").columns:
        labels = sorted(df[col].unique())
        codebook[col] = {str(i): label for i, label in enumerate(labels)}
        df[col] = df[col].map({label: i for i, label in enumerate(labels)})

    # 7. Montos facturados negativos
    stats["negativos_eliminados"] = int((df["Billing Amount"] < 0).sum())
    df = df[df["Billing Amount"] >= 0]

    df = df[COLUMN_ORDER].reset_index(drop=True)
    stats["salida"] = len(df)
    return df, codebook, stats


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--codebook", default=None, help="ruta JSON con el mapeo código -> etiqueta")
    args = parser.parse_args()

    df_in = pd.read_csv(args.input)
    df_out, codebook, stats = clean(df_in)
    df_out.to_csv(args.output, index=False)
    if args.codebook:
        with open(args.codebook, "w", encoding="utf-8") as f:
            json.dump(codebook, f, ensure_ascii=False, indent=2)

    print(f"Filas de entrada:            {stats['entrada']:,}")
    print(f"Sin duplicados exactos:      {stats['sin_duplicados']:,}")
    print(f"Montos negativos eliminados: {stats['negativos_eliminados']:,}")
    print(f"Filas de salida:             {stats['salida']:,}")
