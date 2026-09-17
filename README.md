# Predicción clínica hospitalaria con Redes Neuronales concurrentes en Go

**CC65 – Programación Concurrente y Distribuida · Trabajo Parcial 2026-20 · UPC**

| Integrante | Código | Rol |
|---|---|---|
| Olivera Álvarez, Lizbeth Teresita | U201616851 | Integrante / redactora del informe |
| Gómez Rubina, Luis David | U20221C621 | Coordinador del grupo |
| Zegarra Garcia, Luis Enrique | U202123273 | Integrante |

Docente: Montalvo García Peter Jonathan

## Caso de uso

Apoyo a la priorización clínica en un hospital (ODS 3 – Salud y bienestar): a partir de los datos de admisión de un
paciente (edad, condición médica, tipo de admisión, medicación, monto facturado, días de estancia, etc.) predecir si el
resultado de su examen será **Normal**, **Anormal** o **Inconcluso** (`Test Results`, 3 clases).

Modelo asignado: **Redes Neuronales** (feedforward), a implementar en **Go puro** con goroutines, channels,
`sync.WaitGroup` y `sync.Mutex`, usando los patrones Worker Pool y Pipeline, con verificación de la lógica de
sincronización en Promela/Spin.

## Dataset

| Etapa | Archivo | Filas | Columnas |
|---|---|---|---|
| Original (Kaggle) | `data/raw/healthcare_dataset.csv` | 55 500 | 15 |
| Limpio | `data/processed/healthcare_clean.csv` | 54 860 | 13 |
| Aumentado | `data/healthcare_synthetic_1M.csv` | 1 050 000 | 14 |

El dataset original es el [Healthcare Dataset](https://www.kaggle.com/datasets/prasad22/healthcare-dataset) de Kaggle:
**sintético**, sin pacientes reales.

### Limpieza (`data_cleaning.py`)

1. Eliminación de 534 filas duplicadas exactas (55 500 → 54 966).
2. Conversión de las fechas de admisión y alta; cálculo de `Length of Stay` (días, truncado a ≥ 0).
3. Variables derivadas `Admission Day of Week`, `Admission Month`, `Admission Year`.
4. Descarte de columnas identificatorias o sin valor predictivo: `Name`, `Doctor`, `Hospital`, `Date of Admission`,
   `Discharge Date`, `Room Number`.
5. Codificación entera alfabética de las columnas de texto. El mapeo código → etiqueta queda en
   `data/processed/codebook.json`.
6. Eliminación de 106 registros con `Billing Amount` negativo (54 966 → 54 860). Son un artefacto del generador
   sintético del dataset (la cola baja de la distribución cruza el cero); no representan ningún caso real.

Los pasos 1–5 reproducen la limpieza hecha por el grupo en el curso de Big Data; el paso 6 se añade en este curso.

### Aumento a más de un millón de registros (`data_augmentation.py`)

Bootstrap estratificado: remuestreo con reemplazo dentro de cada combinación `Medical Condition × Admission Type`,
en la misma proporción en que aparece en el dataset limpio, más ruido controlado para evitar filas idénticas
(edad ±1, días de estancia ±1, monto facturado × N(1, 0.04)). Semilla fija → resultado 100 % reproducible.

### Reproducir

```bash
pip install -r requirements.txt
python data_cleaning.py --input data/raw/healthcare_dataset.csv --output data/processed/healthcare_clean.csv --codebook data/processed/codebook.json
python data_augmentation.py --input data/processed/healthcare_clean.csv --output data/healthcare_synthetic_1M.csv --target 1050000 --seed 42
```

## Estructura del repositorio

```
data/
  raw/healthcare_dataset.csv          # original de Kaggle
  processed/healthcare_clean.csv      # dataset limpio (entrada del aumento)
  processed/codebook.json             # mapeo código -> etiqueta de las columnas codificadas
  healthcare_synthetic_1M.csv         # dataset aumentado (entrada del entrenamiento)
data_cleaning.py
data_augmentation.py
requirements.txt
```

## Flujo de trabajo (GitFlow)

- `main`: versión estable de cada entregable.
- `develop`: integración del trabajo del grupo.
- `feature/*`: una rama por funcionalidad, fusionada a `develop` mediante pull request
  (`feature/data-cleaning`, `feature/data-augmentation`, `feature/promela-model`, `feature/go-concurrent-nn`, …).

## Entregables

1. **PC1 (semana 3)** – Investigación bibliográfica, caso de uso, dataset limpio y aumentado. ← *este estado*
2. **PC2 (semana 5)** – Modelo en Promela, implementación secuencial y concurrente en Go, speedup.
3. **TP (semana 7)** – Verificación formal en Spin, informe de análisis con IA, conclusiones.

## Fuentes

- Dataset original: Prasad Patil, *Healthcare Dataset*, Kaggle. https://www.kaggle.com/datasets/prasad22/healthcare-dataset
- Limpieza base (pasos 1–5) tomada del proyecto del curso de Big Data del grupo:
  https://github.com/SofiaGMiranda/BIG-DATA-FINAL-PROYECT-2025-01
