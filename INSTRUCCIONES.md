# Qué sube cada integrante para la PC2 (una rama por persona)

| Integrante | Rama | Contenido |
|---|---|---|
| Luis Enrique Zegarra | `feature/go-red-neuronal` | Programas Go (secuencial y concurrente), benchmark, resultados, figuras y evidencias — ya está en su máquina |
| Luis David Gómez | `feature/promela-model` | Modelo en Promela y salidas de la verificación con Spin |
| Lizbeth Olivera | `feature/readme-pc2` | README actualizado para el Entregable 2 y guía de ejecución |

Orden recomendado: primero se fusiona `feature/go-red-neuronal` a `develop`; después Luis David y Lizbeth crean sus ramas
**desde ese `develop` actualizado** (así el README y el `.gitignore` ya conocen las carpetas nuevas y no hay conflictos).
Al final, `develop → main`.

---

## Luis David Gómez — rama `feature/promela-model`

Archivos que recibe (carpeta `para-companeros/LuisDavid/`), con el mismo destino en el repositorio:

| Archivo | Destino en el repo |
|---|---|
| `promela/entrenamiento.pml` | `promela/entrenamiento.pml` |
| `promela/resultados/sin_semaforo_pan.txt` | `promela/resultados/` |
| `promela/resultados/sin_semaforo_contraejemplo.txt` | `promela/resultados/` |
| `promela/resultados/con_semaforo_pan.txt` | `promela/resultados/` |
| `promela/resultados/con_semaforo_simulacion.txt` | `promela/resultados/` |

Comandos (PowerShell, dentro de la carpeta del repositorio clonado):

```powershell
git checkout develop
git pull
git checkout -b feature/promela-model
# copiar la carpeta promela\ recibida a la raíz del repositorio
git add promela
git commit -m "Añadir el modelo en Promela del entrenamiento concurrente y su verificación con Spin."
git push -u origin feature/promela-model
```

Luego, en GitHub: **Compare & pull request** → base `develop` → Create pull request.

Opcional, si tiene Spin en su máquina (WSL o Linux): reproducir la verificación y guardar capturas de pantalla como
`docs/evidencia/captura_pan_sin_semaforo.png`, `captura_contraejemplo.png` y `captura_pan_con_semaforo.png`, añadiéndolas
al mismo commit (`git add docs/evidencia`):

```bash
cd promela
spin -a -DSIN_SEMAFORO entrenamiento.pml && gcc pan.c -o pan && ./pan     # assertion violated
spin -t -p -g -DSIN_SEMAFORO entrenamiento.pml                            # contraejemplo
spin -a entrenamiento.pml && gcc pan.c -o pan && ./pan                    # errors: 0
```

Como coordinador, además adjunta en el aula virtual el reporte **CC65-Participación-202620**.

---

## Lizbeth Olivera — rama `feature/readme-pc2`

Archivos que recibe (carpeta `para-companeros/Lizbeth/`):

| Archivo | Destino en el repo |
|---|---|
| `README.md` | `README.md` (reemplaza al actual) |
| `docs/GUIA-EJECUCION.md` | `docs/GUIA-EJECUCION.md` |

Comandos:

```powershell
git checkout develop
git pull
git checkout -b feature/readme-pc2
# copiar README.md a la raíz (sobrescribir) y GUIA-EJECUCION.md dentro de docs\
git add README.md docs/GUIA-EJECUCION.md
git commit -m "Actualizar el README y añadir la guía de ejecución para el Entregable 2."
git push -u origin feature/readme-pc2
```

Luego el pull request a `develop`. Opcional: una captura de las ramas y pull requests en GitHub como
`docs/evidencia/captura_github.png` en el mismo commit.

---

## Cierre (Luis Enrique)

1. Fusionar los dos PR anteriores a `develop` y luego abrir y fusionar `develop → main`.
2. Antes de hacer `git pull` de `develop` en tu máquina, borrar la carpeta local `promela\` (está sin seguimiento; la
   versión que llega desde el PR de Luis David es idéntica).
3. Cada integrante sube al aula virtual su `CC65-PC2-202620-<código>.docx`.
