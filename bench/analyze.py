"""Resume bench/results.csv: media recortada al 10 %, speedup, eficiencia, recursos y figuras. Uso: python bench/analyze.py"""
import csv, json, os, statistics as st
from collections import defaultdict

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CSV = os.path.join(RAIZ, "bench", "results.csv")
EVID = os.path.join(RAIZ, "docs", "evidencia")
os.makedirs(EVID, exist_ok=True)

AZUL, NARANJA, GRIS, INK, INK2, GRID, SURF = "#2a78d6", "#eb6834", "#898781", "#0b0b0b", "#52514e", "#e1e0d9", "#fcfcfb"
plt.rcParams.update({
    "font.family": "DejaVu Sans", "font.size": 11, "axes.edgecolor": "#c3c2b7", "axes.labelcolor": INK2,
    "xtick.color": INK2, "ytick.color": INK2, "axes.titlecolor": INK, "axes.titleweight": "bold",
    "axes.spines.top": False, "axes.spines.right": False, "axes.grid": True, "grid.color": GRID,
    "grid.linewidth": 0.8, "axes.axisbelow": True, "figure.facecolor": SURF, "axes.facecolor": SURF,
    "figure.dpi": 150,
})

filas = defaultdict(list)
recursos = {}
with open(CSV, encoding="utf-8") as f:
    for r in csv.DictReader(f):
        k = (r["modo"], int(r["workers"]))
        filas[k].append((int(r["repeticion"]), int(r["tiempo_ms"])))
        recursos[k] = dict(cpu=float(r["cpu_ms_entrenamiento"]), wall=float(r["wall_ms_entrenamiento"]),
                           pico=float(r["pico_mb"]), cpus=int(r["cpus"]), epocas=int(r["epocas"]))

def media_recortada(v, frac=0.10):
    v = sorted(v); k = int(round(len(v) * frac))
    return st.mean(v[k:len(v) - k]) if len(v) - 2 * k > 0 else st.mean(v)

def perdida_final(modo, w):
    ruta = os.path.join(RAIZ, "bench", "salidas", f"{modo}_w{w}.txt")
    valor = None
    if os.path.exists(ruta):
        for linea in open(ruta, encoding="utf-8", errors="replace"):
            if "pérdida = " in linea:
                valor = float(linea.split("pérdida = ")[1].split()[0])
    return valor

resumen = {}
for k, lista in filas.items():
    lista.sort()
    medidas = [t for rep, t in lista if rep > 1]
    if not medidas:
        medidas = [t for _, t in lista]
    rc = recursos[k]
    resumen[k] = dict(
        n=len(medidas), crudos=medidas, media_recortada=media_recortada(medidas), mediana=st.median(medidas),
        desv=st.pstdev(medidas) if len(medidas) > 1 else 0.0, minimo=min(medidas), maximo=max(medidas),
        nucleos_usados=rc["cpu"] / rc["wall"] if rc["wall"] else 0.0,
        pct_cpu_maquina=100 * rc["cpu"] / rc["wall"] / rc["cpus"] if rc["wall"] else 0.0,
        pico_mb=rc["pico"], cpus=rc["cpus"], epocas=rc["epocas"], perdida_final=perdida_final(*k),
    )

t_seq = resumen[("secuencial", 0)]["media_recortada"]
ws = sorted(w for (m, w) in resumen if m == "concurrente")
for w in ws:
    d = resumen[("concurrente", w)]
    d["speedup"] = t_seq / d["media_recortada"]
    d["eficiencia"] = 100 * d["speedup"] / w
resumen[("secuencial", 0)]["speedup"], resumen[("secuencial", 0)]["eficiencia"] = 1.0, 100.0

equilibrio = ws[-1]
for a, b in zip(ws, ws[1:]):
    sa, sb = resumen[("concurrente", a)]["speedup"], resumen[("concurrente", b)]["speedup"]
    if (sb / sa - 1) < 0.05:
        equilibrio = a
        break
mejor_w = max(ws, key=lambda w: resumen[("concurrente", w)]["speedup"])
caida = next((b for a, b in zip(ws, ws[1:]) if a >= mejor_w and resumen[("concurrente", b)]["speedup"] < resumen[("concurrente", a)]["speedup"]), None)

def fila(nombre, d):
    pf = f"{d['perdida_final']:.4f}" if d.get("perdida_final") is not None else "—"
    return (f"| {nombre} | {d['n']} | {d['media_recortada']/1000:.3f} | {d['mediana']/1000:.3f} | {d['desv']/1000:.3f} | "
            f"{d['minimo']/1000:.3f} | {d['maximo']/1000:.3f} | {d['speedup']:.2f}× | {d['eficiencia']:.0f} % | "
            f"{d['nucleos_usados']:.1f} | {d['pico_mb']:.0f} | {pf} |")

lineas = ["| Configuración | n | Media recortada (s) | Mediana (s) | Desv. est. (s) | Mín (s) | Máx (s) | Speedup | Eficiencia | Núcleos usados | RAM pico (MB) | Pérdida final |",
          "|---|---|---|---|---|---|---|---|---|---|---|---|", fila("Secuencial (1 hilo)", resumen[("secuencial", 0)])]
lineas += [fila(f"Concurrente, W = {w}", resumen[("concurrente", w)]) for w in ws]
ep = resumen[("secuencial", 0)]["epocas"]
lineas.append("")
lineas.append(f"Épocas por corrida: {ep}. n = medidas por configuración tras descartar 1 calentamiento. Media recortada al 10 % "
              f"(se descartan la mejor y la peor de 10). Speedup = T_secuencial / T_concurrente. Núcleos usados = CPU ms / wall ms "
              f"durante el entrenamiento (máquina de {resumen[('secuencial', 0)]['cpus']} núcleos lógicos).")
lineas.append(f"Mejor speedup: {resumen[('concurrente', mejor_w)]['speedup']:.2f}× con W = {mejor_w}. "
              f"Punto de equilibrio: W = {equilibrio} (a partir de ahí la mejora marginal es < 5 %). "
              + (f"El speedup empieza a disminuir a partir de W = {caida}." if caida else "El speedup no llegó a disminuir en el rango probado."))
with open(os.path.join(RAIZ, "bench", "tabla.md"), "w", encoding="utf-8") as f:
    f.write("\n".join(lineas) + "\n")

salida_json = {"t_secuencial_ms": t_seq, "epocas": ep, "cpus": resumen[("secuencial", 0)]["cpus"],
               "equilibrio": equilibrio, "mejor_w": mejor_w, "caida": caida,
               "configs": [dict(modo=m, workers=w, **{k: v for k, v in d.items()}) for (m, w), d in sorted(resumen.items())]}
with open(os.path.join(RAIZ, "bench", "resumen.json"), "w", encoding="utf-8") as f:
    json.dump(salida_json, f, ensure_ascii=False, indent=1)

etiquetas = ["Sec."] + [str(w) for w in ws]
x = list(range(len(etiquetas)))
tiempos = [t_seq / 1000] + [resumen[("concurrente", w)]["media_recortada"] / 1000 for w in ws]
desv = [resumen[("secuencial", 0)]["desv"] / 1000] + [resumen[("concurrente", w)]["desv"] / 1000 for w in ws]
speed = [resumen[("concurrente", w)]["speedup"] for w in ws]
efic = [resumen[("concurrente", w)]["eficiencia"] for w in ws]

def guardar(fig, nombre):
    ruta = os.path.join(EVID, nombre)
    fig.savefig(ruta, bbox_inches="tight", facecolor=SURF)
    plt.close(fig)
    print("figura:", os.path.relpath(ruta, RAIZ))

fig, ax = plt.subplots(figsize=(9, 4.6))
colores = [NARANJA] + [AZUL] * len(ws)
ax.bar(x, tiempos, color=colores, width=0.62, yerr=desv, capsize=3, error_kw=dict(ecolor=INK2, lw=1))
for xi, t in zip(x, tiempos):
    ax.text(xi, t + max(tiempos) * 0.015, f"{t:.2f} s", ha="center", va="bottom", fontsize=9, color=INK2)
ax.axhline(t_seq / 1000, color=NARANJA, lw=1, ls=(0, (4, 3)))
ax.set_xticks(x, etiquetas); ax.set_xlabel("Número de workers (goroutines)"); ax.set_ylabel("Tiempo de entrenamiento (s)")
ax.set_title(f"Tiempo de entrenamiento por configuración ({ep} épocas, media recortada de 10 corridas)")
ax.grid(axis="x", visible=False)
guardar(fig, "fig1_tiempos.png")

cpus = resumen[("secuencial", 0)]["cpus"]
fis = cpus // 2
ideal_w = [w for w in ws if w <= cpus]
TEAL = "#1baf7a"

def eje_workers(ax):
    ax.set_xscale("log", base=2)
    ax.set_xticks(ws); ax.set_xticklabels([str(w) for w in ws]); ax.minorticks_off()
    ax.set_xlabel("Número de workers (goroutines), escala log₂")
    ax.axvline(fis, color=GRIS, lw=1, ls=":")
    ax.axvline(equilibrio, color=TEAL, lw=1.5, ls=(0, (4, 3)))
    if max(ws) > cpus:
        ax.axvspan(cpus, max(ws) * 1.15, color=GRIS, alpha=0.08, lw=0)

fig, ax = plt.subplots(figsize=(9, 4.8))
ax.plot(ideal_w, ideal_w, color=GRIS, lw=1.5, ls=(0, (4, 3)), label="Ideal (speedup = W)")
ax.plot(ws, speed, color=AZUL, lw=2, marker="o", ms=7, label="Medido")
for w, s in zip(ws, speed):
    ax.annotate(f"{s:.1f}×", (w, s), textcoords="offset points", xytext=(0, 9), ha="center", fontsize=9, color=INK2)
eje_workers(ax)
tope = max(max(speed), cpus) * 1.05
ax.text(fis * 1.03, tope * 0.97, f"{fis} núcleos físicos", color=INK2, fontsize=9, va="top")
ax.text(equilibrio * 1.03, tope * 0.84, f"punto de equilibrio (W = {equilibrio})", color=TEAL, fontsize=9, va="top")
if max(ws) > cpus:
    ax.text(cpus * 1.03, tope * 0.71, f"sobresuscripción (W > {cpus} hilos lógicos)", color=INK2, fontsize=9, va="top")
ax.set_ylim(0, tope)
ax.set_ylabel("Speedup = T secuencial / T concurrente")
ax.set_title("Speedup del entrenamiento concurrente frente al secuencial")
ax.legend(frameon=False, loc="upper left")
guardar(fig, "fig2_speedup.png")

fig, ax = plt.subplots(figsize=(9, 4.4))
ax.plot(ws, efic, color=AZUL, lw=2, marker="o", ms=7)
for w, e in zip(ws, efic):
    ax.annotate(f"{e:.0f} %", (w, e), textcoords="offset points", xytext=(0, 9), ha="center", fontsize=9, color=INK2)
ax.axhline(100, color=GRIS, lw=1, ls=(0, (4, 3)))
eje_workers(ax)
ax.set_ylim(0, 115)
ax.text(equilibrio * 1.03, 112, f"punto de equilibrio (W = {equilibrio})", color=TEAL, fontsize=9, va="top")
ax.set_ylabel("Eficiencia = speedup / W (%)")
ax.set_title("Eficiencia del paralelismo: cuánto aporta cada worker adicional")
guardar(fig, "fig3_eficiencia.png")

fig, ax = plt.subplots(figsize=(9, 4.6))
for xi, k in zip(x, [("secuencial", 0)] + [("concurrente", w) for w in ws]):
    d = resumen[k]
    ax.scatter([xi] * len(d["crudos"]), [v / 1000 for v in d["crudos"]], s=18, color=GRIS, alpha=0.7, zorder=2)
    ax.scatter([xi], [d["media_recortada"] / 1000], s=70, color=AZUL, marker="D", zorder=3)
ax.scatter([], [], s=18, color=GRIS, label="Corrida individual (10 por configuración)")
ax.scatter([], [], s=70, color=AZUL, marker="D", label="Media recortada al 10 %")
ax.set_xticks(x, etiquetas); ax.set_xlabel("Número de workers (goroutines)"); ax.set_ylabel("Tiempo de entrenamiento (s)")
ax.set_title("Dispersión de las corridas y media recortada por configuración")
ax.legend(frameon=False); ax.grid(axis="x", visible=False)
guardar(fig, "fig4_dispersion.png")

nuc = [resumen[("secuencial", 0)]["nucleos_usados"]] + [resumen[("concurrente", w)]["nucleos_usados"] for w in ws]
mem = [resumen[("secuencial", 0)]["pico_mb"]] + [resumen[("concurrente", w)]["pico_mb"] for w in ws]
fig, (a1, a2) = plt.subplots(1, 2, figsize=(11, 4.4))
a1.bar(x, nuc, color=colores, width=0.62)
for xi, v in zip(x, nuc):
    a1.text(xi, v + max(nuc) * 0.015, f"{v:.1f}", ha="center", va="bottom", fontsize=9, color=INK2)
a1.set_xticks(x, etiquetas); a1.set_xlabel("Workers"); a1.set_ylabel("Núcleos lógicos usados (CPU ms / wall ms)")
a1.set_title("Uso de CPU durante el entrenamiento"); a1.grid(axis="x", visible=False)
a2.bar(x, mem, color=colores, width=0.62)
for xi, v in zip(x, mem):
    a2.text(xi, v + max(mem) * 0.015, f"{v:.0f}", ha="center", va="bottom", fontsize=9, color=INK2)
a2.set_xticks(x, etiquetas); a2.set_xlabel("Workers"); a2.set_ylabel("Memoria pico del proceso (MB)")
a2.set_title("Memoria utilizada"); a2.grid(axis="x", visible=False)
guardar(fig, "fig5_recursos.png")

print("\n".join(lineas))
