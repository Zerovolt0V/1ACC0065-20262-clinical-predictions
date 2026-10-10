| Configuración | n | Media recortada (s) | Mediana (s) | Desv. est. (s) | Mín (s) | Máx (s) | Speedup | Eficiencia | Núcleos usados | RAM pico (MB) | Pérdida final |
|---|---|---|---|---|---|---|---|---|---|---|---|
| Secuencial (1 hilo) | 10 | 8.350 | 8.346 | 0.088 | 8.201 | 8.492 | 1.00× | 100 % | 1.0 | 649 | 1.0523 |
| Concurrente, W = 1 | 10 | 7.913 | 7.841 | 0.263 | 7.402 | 8.392 | 1.06× | 106 % | 1.0 | 649 | 1.0523 |
| Concurrente, W = 2 | 10 | 4.805 | 4.809 | 0.139 | 4.530 | 5.054 | 1.74× | 87 % | 2.0 | 652 | 1.0664 |
| Concurrente, W = 4 | 10 | 2.856 | 2.847 | 0.082 | 2.632 | 2.962 | 2.92× | 73 % | 3.9 | 652 | 1.0804 |
| Concurrente, W = 8 | 10 | 1.587 | 1.583 | 0.146 | 1.114 | 1.636 | 5.26× | 66 % | 7.6 | 657 | 1.0900 |
| Concurrente, W = 12 | 10 | 1.145 | 1.190 | 0.157 | 0.790 | 1.218 | 7.29× | 61 % | 11.2 | 661 | 1.0941 |
| Concurrente, W = 16 | 10 | 0.920 | 0.936 | 0.064 | 0.782 | 0.961 | 9.08× | 57 % | 14.8 | 667 | 1.0961 |
| Concurrente, W = 24 | 10 | 0.710 | 0.710 | 0.018 | 0.690 | 0.757 | 11.77× | 49 % | 21.9 | 675 | 1.0980 |
| Concurrente, W = 32 | 10 | 0.828 | 0.819 | 0.044 | 0.781 | 0.898 | 10.09× | 32 % | 18.6 | 686 | 1.0989 |
| Concurrente, W = 48 | 10 | 0.795 | 0.799 | 0.030 | 0.748 | 0.856 | 10.50× | 22 % | 19.5 | 707 | 1.1000 |
| Concurrente, W = 64 | 10 | 0.788 | 0.789 | 0.025 | 0.743 | 0.824 | 10.60× | 17 % | 20.0 | 727 | 1.1007 |
| Concurrente, W = 96 | 10 | 0.760 | 0.762 | 0.013 | 0.733 | 0.777 | 10.98× | 11 % | 21.1 | 762 | 1.1019 |
| Concurrente, W = 128 | 10 | 0.750 | 0.751 | 0.011 | 0.728 | 0.760 | 11.14× | 9 % | 21.6 | 792 | 1.1029 |

Épocas por corrida: 3. n = medidas por configuración tras descartar 1 calentamiento. Media recortada al 10 % (se descartan la mejor y la peor de 10). Speedup = T_secuencial / T_concurrente. Núcleos usados = CPU ms / wall ms durante el entrenamiento (máquina de 24 núcleos lógicos).
Mejor speedup: 11.77× con W = 24. Punto de equilibrio: W = 24 (a partir de ahí la mejora marginal es < 5 %). El speedup empieza a disminuir a partir de W = 32.
