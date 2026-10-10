# Prompt estructurado para el análisis del código con IA

**Modelo seleccionado:** Claude Opus 5 (`claude-opus-5`), ejecutado desde Claude Code (Anthropic) el 10 de octubre de 2026,
con acceso de lectura al repositorio en la rama `develop` (commit `0e3eaed`).
**Resultado:** [`ANALISIS-IA.md`](ANALISIS-IA.md).

El prompt se diseñó con cinco partes (rol, contexto, alcance, dimensiones de análisis y formato de salida) para que la
respuesta fuera verificable: cada hallazgo debe citar archivo y línea, asignar una severidad y proponer una corrección concreta.

---

## Prompt

```text
ROL
Actúa como revisor senior de software especializado en Go y en programación concurrente. Vas a auditar el código
de un proyecto académico (curso CC65 – Programación Concurrente y Distribuida, UPC) y producir un informe técnico
de brechas (GAPs). Sé concreto y verificable: cada hallazgo debe citar archivo y línea, explicar el riesgo real
y proponer la corrección mínima. No inventes problemas: si una práctica es aceptable para el contexto, dilo.

CONTEXTO DEL PROYECTO
- Caso de uso: priorización clínica hospitalaria. Una red neuronal feedforward (52 → 64 → 3) predice si el
  examen de un paciente será Normal, Anormal o Inconcluso, sobre un dataset sintético de 1 050 000 registros.
- Restricción del curso: Go puro, sin librerías de terceros, usando goroutines, sync.WaitGroup y sync.Mutex,
  con los patrones Worker Pool y Pipeline; cada programa en un solo archivo, con lo visto en clase (sin channels,
  sin paquetes propios).
- Algoritmo concurrente: SPMD por época. W goroutines entrenan particiones disjuntas con copias locales de los
  pesos; al final de cada época suman sus pesos en acumuladores compartidos dentro de una sección crítica (Mutex);
  el principal espera con WaitGroup (barrera) y promedia. La predicción usa un worker pool sobre el 20 % de prueba.
- El modelo de sincronización fue verificado en Promela/Spin (exclusión mutua y ausencia de deadlock).
- Resultados: speedup máximo 11.77× con 24 goroutines (24 hilos lógicos); cae con sobresuscripción.

ALCANCE (archivos a revisar)
- secuencial/secuencial.go      entrenamiento en un hilo
- concurrente/concurrente.go    entrenamiento SPMD + worker pool de predicción
- bench/run_bench.ps1           benchmark: un proceso por configuración, muestreo de CPU y memoria
- bench/analyze.py              media recortada, speedup, eficiencia, figuras
- promela/entrenamiento.pml     modelo Promela de la sincronización
- .gitignore, README.md, docs/GUIA-EJECUCION.md

DIMENSIONES DE ANÁLISIS
1. Calidad de código: manejo de errores, validación de entradas, duplicación, legibilidad, formato (gofmt),
   nombres, comentarios, estructura, pruebas automatizadas.
2. Seguridad: entradas no confiables (argumentos, CSV), rutas, escritura de archivos, dependencias, fugas de
   información, límites de recursos (memoria/CPU) ante parámetros extremos.
3. Patrones de concurrencia: corrección (carreras, deadlocks, uso de WaitGroup/Mutex), granularidad de la
   sección crítica, false sharing, balance de carga, sobresuscripción, patrones faltantes (channels/pipeline),
   reproducibilidad.
4. Rendimiento y escalabilidad: costo de la agregación, memoria, E/S, oportunidades de mejora medibles.
5. Mantenibilidad y reproducibilidad: go.mod, versiones, scripts, documentación, CI.

EVIDENCIA ADICIONAL QUE DEBES CONSIDERAR
- Salida de `go vet` sobre ambos programas.
- Salida de `gofmt -l`.
- Salida de `go run -race concurrente/concurrente.go 4 1 1`.

FORMATO DE SALIDA (Markdown)
1. Resumen ejecutivo (5 líneas): estado general y los 3 hallazgos más importantes.
2. Tabla de hallazgos con columnas: ID · Dimensión · Severidad (Alta/Media/Baja) · Ubicación (archivo:línea) ·
   Descripción · Riesgo · Recomendación.
3. Detalle de cada hallazgo de severidad Alta con un fragmento de código y la corrección propuesta.
4. Lo que está bien (prácticas correctas que conviene mantener), con evidencia.
5. Plan de remediación priorizado (qué corregir primero y esfuerzo estimado).
6. Conclusión sobre la madurez del código para un Entregable 4 hipotético.
```
