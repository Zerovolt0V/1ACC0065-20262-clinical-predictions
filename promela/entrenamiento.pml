/* entrenamiento.pml
   Modelo en Promela de la lógica de sincronización del entrenamiento concurrente
   (SPMD) de la red neuronal:
     - W workers (goroutines) entrenan su partición con pesos LOCALES.
     - Al final de cada época, cada worker suma sus pesos en un acumulador
       COMPARTIDO: sección crítica protegida con un semáforo (= sync.Mutex).
     - El proceso principal espera a que todos terminen (terminados == W, que
       equivale a sync.WaitGroup.Wait), promedia, y libera la siguiente época.


*/

#define W 3      
#define E 2      

byte sem = 1;       
byte critical = 0;   
byte terminados = 0; 
byte epoca = 0;      
int  suma = 0;       

inline wait(s) {
    atomic {
        s > 0;
        s--
    }
}
inline signal(s) {
    s++
}

proctype Trabajador(byte id) {
    byte mi_epoca = 0;
    int  peso_local;
    int  temp;
    do
    :: mi_epoca < E ->
        peso_local = id;                 
#ifndef SIN_SEMAFORO
        wait(sem);                       
#endif
        critical++;
        assert(critical <= 1);           
        temp = suma;                     
        temp = temp + peso_local;        
        suma = temp;
        critical--;
#ifndef SIN_SEMAFORO
        signal(sem);                     
#endif
        atomic { terminados++ };         
        mi_epoca++;
        (epoca == mi_epoca);             
    :: else ->
        break
    od
}

init {
    byte e = 0;
    atomic {
        run Trabajador(1);
        run Trabajador(2);
        run Trabajador(3)
    }
    do
    :: e < E ->
        (terminados == W);               
        assert(suma == 1 + 2 + 3);       
        printf("Epoca %d: suma = %d, promedio = %d\n", e + 1, suma, suma / W);
        suma = 0;
        terminados = 0;
        e++;
        epoca = e                      
    :: else ->
        break
    od;
    assert(epoca == E);
    printf("Entrenamiento terminado: %d epocas, sin condiciones de carrera\n", epoca)
}
