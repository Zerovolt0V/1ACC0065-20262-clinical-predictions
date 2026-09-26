# Guía de ejecución — Entregable 2 (PC2)

Cómo ejecutar los programas de este entregable, qué imprimen y cómo reproducir el benchmark de speedup.
Todo se corre desde la raíz del repositorio. Requisitos: Go 1.22 o superior en Windows; Spin y gcc en WSL (Ubuntu).

## 1. Programa secuencial

```powershell
go run secuencial/secuencial.go 3 1
```

Argumentos: `[épocas] [repeticiones]` (por defecto 2 y 1). Un solo hilo, sin goroutines.

Salida:

1. `Datos cargados en … s | entrenamiento: 840000 filas | prueba: 210000 filas | entradas: 52 | CPUs: 24`
2. Por cada repetición, la pérdida de cada época y la línea `TIEMPO_MS=…` (tiempo de entrenamiento en milisegundos;
   la carga del CSV no se cronometra).
3. `=== Predicción sobre 210000 pacientes de prueba ===`: exactitud y conteo por clase.
4. `=== Cola de priorización ===`: los 10 pacientes con mayor probabilidad de examen **Anormal**, con sus datos
   decodificados. La cola completa queda en `predicciones_secuencial.csv`.

## 2. Programa concurrente

```powershell
go run concurrente/concurrente.go 24 3 1
```

Argumentos: `[workers] [épocas] [repeticiones]` (por defecto: número de CPUs, 2 y 1).

- Con `workers = 1` la pérdida final es **idéntica** a la del secuencial (misma semilla, mismo orden): así se valida que
  ambos programas hacen el mismo cómputo.
- Con `workers = 24` (un worker por hilo lógico; máximo y punto de equilibrio en el equipo de pruebas) el entrenamiento
  tarda ≈ 0.7 s por corrida de 3 épocas frente a ≈ 8.4 s del secuencial (speedup ≈ 11.8×). Con más workers que hilos
  lógicos el tiempo empeora.

Además de lo anterior imprime `Worker pool de predicción: W workers, N aciertos de 210000`.

## 3. Benchmark completo (secuencial + W ∈ {1, 2, 4, 8, 12, 16, 24, 32, 48, 64, 96, 128})

```powershell
Remove-Item bench\results.csv, bench\salidas\* -Force -ErrorAction SilentlyContinue   # barrido limpio
powershell -ExecutionPolicy Bypass -File bench/run_bench.ps1 -Workers "1,2,4,8,12,16,24,32,48,64,96,128"   # ≈ 7 min; no usar la PC
powershell -ExecutionPolicy Bypass -File bench/run_bench.ps1 -Workers 24 -SinSecuencial   # un solo W, se añade al CSV (demo)
python bench/analyze.py                                                  # tabla, resumen y figuras
```

La lista de workers va entre comillas y separada por comas. Sin `-SinSecuencial` el script reinicia `bench/results.csv`;
con `-SinSecuencial` añade filas al archivo existente.

`run_bench.ps1` lanza **un proceso independiente por configuración** (1 calentamiento + 10 repeticiones cada uno), muestrea
CPU y memoria cada 100 ms y escribe `bench/results.csv` (una fila por repetición) y `bench/salidas/*.txt` (salida completa
de cada proceso). `analyze.py` descarta el calentamiento, calcula la **media recortada al 10 %** (quita la mejor y la peor de
10), mediana, desviación estándar, speedup, eficiencia y pérdida final, identifica el punto de equilibrio y desde qué W empieza
a bajar el speedup, y genera `bench/tabla.md`, `bench/resumen.json` y las figuras en `docs/evidencia/` (carpeta local, no
versionada: `fig1_tiempos.png`, `fig2_speedup.png`, `fig3_eficiencia.png`, `fig4_dispersion.png`, `fig5_recursos.png`).

## 4. Modelo Promela (WSL)

```bash
cd /mnt/e/UPC/PCD/TP/1ACC0065-20262-clinical-predictions/promela
spin -a -DSIN_SEMAFORO entrenamiento.pml && gcc pan.c -o pan && ./pan   # SIN semáforo: assertion violated (hay carrera)
spin -t -p -g -DSIN_SEMAFORO entrenamiento.pml                          # contraejemplo paso a paso
spin -a entrenamiento.pml && gcc pan.c -o pan && ./pan                  # CON semáforo: errors: 0
spin -g entrenamiento.pml                                               # simulación aleatoria
```

Las salidas de referencia están en `promela/resultados/`.

## 5. Cómo leer los resultados

| Concepto | Dónde | Qué significa |
|---|---|---|
| `TIEMPO_MS` | salida de los programas | milisegundos del entrenamiento de una repetición |
| Media recortada | `bench/tabla.md` | promedio de las 8 corridas centrales de 10 |
| Speedup | `bench/tabla.md` | T_secuencial / T_concurrente(W) |
| Eficiencia | `bench/tabla.md` | speedup / W |
| Núcleos usados | `bench/tabla.md` | CPU ms / wall ms del proceso durante el entrenamiento |
| Pérdida final | `bench/tabla.md` | pérdida de entrenamiento tras la última época (sube con W: trade-off de convergencia) |
| Punto de equilibrio | `bench/resumen.json` (`equilibrio`) | primer W a partir del cual la mejora marginal es < 5 % |
| Caída | `bench/resumen.json` (`caida`) | primer W en el que el speedup baja respecto al anterior (sobresuscripción) |
