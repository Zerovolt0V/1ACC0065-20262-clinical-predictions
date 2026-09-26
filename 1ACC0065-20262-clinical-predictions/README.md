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
resultado de su examen será **Normal**, **Anormal** o **Inconcluso** (`Test Results`, 3 clases), y ordenar a los pacientes
en una **cola de priorización** según su probabilidad de examen anormal.

Modelo asignado: **Redes Neuronales** (feedforward 52 → 64 → 3), implementadas en **Go puro** con goroutines,
`sync.WaitGroup` y `sync.Mutex`, con el patrón Worker Pool, y con la lógica de sincronización modelada en Promela y
verificada con Spin.

## Entregable 2 — programas en Go, Promela y speedup

### Ejecutar

```powershell
go run secuencial/secuencial.go 3 1          # secuencial: [épocas] [repeticiones]
go run concurrente/concurrente.go 24 3 1     # concurrente: [workers] [épocas] [repeticiones]
```

Ambos cargan el dataset de 1 050 000 registros, entrenan la red (80 % de las filas), predicen el 20 % restante y muestran
la cola de priorización (los 10 pacientes con mayor probabilidad de examen anormal); la cola completa se guarda en
`predicciones_*.csv`. Cada programa está en **un solo archivo** y usa únicamente lo visto en clase.

### Algoritmo concurrente (SPMD con promedio por época)

W goroutines entrenan cada una una partición del dataset con una **copia local** de los pesos; al terminar la época suman
sus pesos en acumuladores compartidos dentro de una **sección crítica** (`sync.Mutex`); el programa principal espera a
todas con `sync.WaitGroup` (barrera), promedia y lanza la siguiente época. La predicción usa un **Worker Pool** de W
goroutines sobre rangos disjuntos del conjunto de prueba.

### Benchmark

```powershell
Remove-Item bench\results.csv, bench\salidas\* -Force -ErrorAction SilentlyContinue      # empezar un barrido limpio
powershell -ExecutionPolicy Bypass -File bench/run_bench.ps1 -Workers "1,2,4,8,12,16,24,32,48,64,96,128"   # ≈ 7 min
python bench/analyze.py                                           # media recortada, speedup, eficiencia, figuras
```

`run_bench.ps1` ejecuta el secuencial y cada número de workers como un proceso independiente (1 calentamiento +
10 repeticiones cada uno) y muestrea la CPU y la memoria del proceso cada 100 ms. `analyze.py` calcula la media
recortada al 10 %, el speedup, la eficiencia y el punto de equilibrio, escribe `bench/tabla.md` y `bench/resumen.json`,
y genera las figuras en `docs/evidencia/` (carpeta local, no versionada).

Resultados en el equipo de pruebas (Ryzen 9 7900X3D, 12 núcleos / 24 hilos lógicos, 3 épocas, media recortada de 10 corridas):

| Configuración | Tiempo | Speedup | Eficiencia |
|---|---|---|---|
| Secuencial | 8.35 s | 1.00× | — |
| Concurrente W = 4 | 2.86 s | 2.92× | 73 % |
| Concurrente W = 8 | 1.59 s | 5.26× | 66 % |
| Concurrente W = 12 | 1.15 s | 7.29× | 61 % |
| Concurrente W = 16 | 0.92 s | 9.08× | 57 % |
| **Concurrente W = 24** | **0.71 s** | **11.77×** | 49 % |
| Concurrente W = 32 | 0.83 s | 10.09× | 32 % |
| Concurrente W = 128 | 0.75 s | 11.14× | 9 % |

El speedup crece hasta **W = 24** (uno por hilo lógico), que es el máximo y el punto de equilibrio; con más goroutines
que hilos (sobresuscripción) cae y ya no se recupera, mientras la memoria y la pérdida por época empeoran. Tabla completa
en `bench/tabla.md`.

### Modelo en Promela (WSL)

```bash
cd promela
spin -a -DSIN_SEMAFORO entrenamiento.pml && gcc pan.c -o pan && ./pan   # sin semáforo: assertion violated
spin -a entrenamiento.pml && gcc pan.c -o pan && ./pan                  # con semáforo: errors: 0, sin deadlocks
```

Salidas de referencia en `promela/resultados/`.

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
secuencial/secuencial.go              # entrenamiento secuencial (1 hilo) + predicción + cola de priorización
concurrente/concurrente.go            # entrenamiento concurrente SPMD (goroutines, WaitGroup, Mutex) + worker pool
promela/entrenamiento.pml             # modelo de la sincronización; -DSIN_SEMAFORO reproduce la carrera
promela/resultados/                   # salidas de spin/pan con y sin semáforo
bench/run_bench.ps1                   # benchmark: un proceso por configuración, CPU y memoria muestreadas
bench/analyze.py                      # media recortada, speedup, eficiencia, figuras
bench/results.csv · tabla.md · resumen.json · salidas/   # resultados del barrido de 1 a 128 workers
docs/GUIA-EJECUCION.md                # guía detallada de ejecución
data/                                 # datasets (raw, limpio, aumentado) y codebook
data_cleaning.py · data_augmentation.py · requirements.txt
```

## Flujo de trabajo (GitFlow)

- `main`: versión estable de cada entregable.
- `develop`: integración del trabajo del grupo.
- `feature/*`: una rama por funcionalidad, fusionada a `develop` mediante pull request.
  - PC1: `feature/data-cleaning` (limpieza y dataset), `feature/dataset-original`, `feature/readme`.
  - PC2: `feature/go-red-neuronal` (programas secuencial y concurrente, benchmark y evidencias),
    `feature/promela-model` (modelo en Promela y verificación con Spin), `feature/readme-pc2` (README y guía de ejecución).

## Entregables

1. **PC1 (semana 3)** – Investigación bibliográfica, caso de uso, dataset limpio y aumentado. ✔
2. **PC2 (semana 5)** – Modelo en Promela, implementación secuencial y concurrente en Go, speedup. ← *este estado*
3. **TP (semana 7)** – Verificación formal en Spin, informe de análisis con IA, conclusiones.

## Fuentes

- Dataset original: Prasad Patil, *Healthcare Dataset*, Kaggle. https://www.kaggle.com/datasets/prasad22/healthcare-dataset
- Limpieza base (pasos 1–5) tomada del proyecto del curso de Big Data del grupo:
  https://github.com/SofiaGMiranda/BIG-DATA-FINAL-PROYECT-2025-01
- Algoritmo SPMD en Go: Kalwarowskyj, D., & Schikuta, E. (2023). *SPMD-based neural network simulation with Golang*. ICCS 2023.
