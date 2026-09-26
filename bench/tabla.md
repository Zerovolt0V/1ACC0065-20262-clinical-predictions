| Configuración | n | Media recortada (s) | Mediana (s) | Desv. est. (s) | Mín (s) | Máx (s) | Speedup | Eficiencia | Núcleos usados | RAM pico (MB) |
|---|---|---|---|---|---|---|---|---|---|---|
| Secuencial (1 hilo) | 10 | 7.353 | 7.366 | 0.050 | 7.257 | 7.446 | 1.00× | 100 % | 1.0 | 649 |
| Concurrente, W = 1 | 10 | 7.357 | 7.317 | 0.109 | 7.179 | 7.580 | 1.00× | 100 % | 1.0 | 648 |
| Concurrente, W = 2 | 10 | 3.720 | 3.717 | 0.035 | 3.667 | 3.787 | 1.98× | 99 % | 2.0 | 651 |
| Concurrente, W = 4 | 10 | 1.928 | 1.925 | 0.011 | 1.908 | 1.941 | 3.81× | 95 % | 4.0 | 649 |
| Concurrente, W = 8 | 10 | 1.226 | 1.062 | 0.276 | 1.037 | 1.715 | 6.00× | 75 % | 7.8 | 656 |
| Concurrente, W = 12 | 10 | 1.027 | 1.022 | 0.148 | 0.804 | 1.193 | 7.16× | 60 % | 11.2 | 664 |
| Concurrente, W = 16 | 10 | 0.788 | 0.789 | 0.021 | 0.764 | 0.839 | 9.33× | 58 % | 14.4 | 666 |
| Concurrente, W = 24 | 10 | 0.764 | 0.765 | 0.018 | 0.742 | 0.808 | 9.62× | 40 % | 20.3 | 681 |

Épocas por corrida: 3. n = medidas por configuración tras descartar 1 calentamiento. Media recortada al 10 % (se descartan la mejor y la peor de 10). Speedup = T_secuencial / T_concurrente. Núcleos usados = CPU ms / wall ms durante el entrenamiento (máquina de 24 núcleos lógicos).
Mejor speedup: 9.62× con W = 24. Punto de equilibrio: W = 16 (a partir de ahí la mejora marginal es < 5 %).
