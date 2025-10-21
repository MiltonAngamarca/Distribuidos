#!/bin/bash

# Script para demostrar race conditions en el sistema de reservas
# Este script envía múltiples peticiones concurrentes al mismo asiento
# para provocar el problema de doble reserva

echo "🎬 DEMOSTRACIÓN DE RACE CONDITIONS"
echo "=================================="
echo ""

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# URLs de los servidores
LOAD_BALANCER="http://localhost:8080"
SERVIDOR_1="http://localhost:8081"
SERVIDOR_2="http://localhost:8082"
SERVIDOR_3="http://localhost:8083"

# Función para verificar si los servidores están activos
check_servers() {
    echo -e "${BLUE}🔍 Verificando estado de los servidores...${NC}"
    
    for i in {1..3}; do
        url="http://localhost:808$i/health"
        if curl -s "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Servidor $i (puerto 808$i) - ACTIVO${NC}"
        else
            echo -e "${RED}❌ Servidor $i (puerto 808$i) - INACTIVO${NC}"
        fi
    done
    echo ""
}

# Función para resetear todos los servidores
reset_servers() {
    echo -e "${YELLOW}🔄 Reseteando todos los servidores...${NC}"
    
    for i in {1..3}; do
        url="http://localhost:808$i/reset"
        curl -s -X POST "$url" > /dev/null 2>&1
    done
    
    echo -e "${GREEN}✅ Servidores reseteados${NC}"
    echo ""
}

# Función para mostrar el estado de un asiento en todos los servidores
show_seat_status() {
    local seat_number=$1
    echo -e "${BLUE}📊 Estado del asiento $seat_number en todos los servidores:${NC}"
    
    for i in {1..3}; do
        url="http://localhost:808$i/asiento/$seat_number"
        response=$(curl -s "$url")
        
        if [ $? -eq 0 ]; then
            disponible=$(echo "$response" | grep -o '"disponible":[^,]*' | cut -d':' -f2)
            cliente=$(echo "$response" | grep -o '"cliente":"[^"]*"' | cut -d'"' -f4)
            servidor=$(echo "$response" | grep -o '"servidor_id":"[^"]*"' | cut -d'"' -f4)
            
            if [ "$disponible" = "true" ]; then
                echo -e "  Servidor $i: ${GREEN}DISPONIBLE${NC}"
            else
                echo -e "  Servidor $i: ${RED}RESERVADO${NC} por '$cliente'"
            fi
        else
            echo -e "  Servidor $i: ${RED}ERROR${NC}"
        fi
    done
    echo ""
}

# Función para reservar asiento de forma concurrente
concurrent_reservation() {
    local seat_number=$1
    local client_name=$2
    local server_port=$3
    
    url="http://localhost:$server_port/reservar"
    
    response=$(curl -s -X POST "$url" \
        -H "Content-Type: application/json" \
        -d "{\"numero\": $seat_number, \"cliente\": \"$client_name\"}")
    
    success=$(echo "$response" | grep -o '"success":[^,]*' | cut -d':' -f2)
    
    if [ "$success" = "true" ]; then
        echo -e "${GREEN}✅ [$client_name] Reserva exitosa en servidor puerto $server_port${NC}"
    else
        error=$(echo "$response" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)
        echo -e "${RED}❌ [$client_name] Error en servidor puerto $server_port: $error${NC}"
    fi
}

# Función principal para demostrar race condition
demonstrate_race_condition() {
    local seat_number=$1
    
    echo -e "${YELLOW}🚀 INICIANDO PRUEBA DE RACE CONDITION${NC}"
    echo -e "${YELLOW}Asiento objetivo: $seat_number${NC}"
    echo ""
    
    # Mostrar estado inicial
    echo -e "${BLUE}📋 Estado inicial:${NC}"
    show_seat_status $seat_number
    
    # Enviar peticiones concurrentes
    echo -e "${YELLOW}⚡ Enviando peticiones concurrentes...${NC}"
    echo ""
    
    # Ejecutar reservas en paralelo
    concurrent_reservation $seat_number "Cliente-A" 8081 &
    concurrent_reservation $seat_number "Cliente-B" 8082 &
    concurrent_reservation $seat_number "Cliente-C" 8083 &
    
    # Esperar a que terminen todas las peticiones
    wait
    
    echo ""
    echo -e "${BLUE}📋 Estado final:${NC}"
    show_seat_status $seat_number
    
    # Verificar si hay inconsistencia
    echo -e "${YELLOW}🔍 Análisis de resultados:${NC}"
    
    reservado_count=0
    for i in {1..3}; do
        url="http://localhost:808$i/asiento/$seat_number"
        response=$(curl -s "$url")
        disponible=$(echo "$response" | grep -o '"disponible":[^,]*' | cut -d':' -f2)
        
        if [ "$disponible" = "false" ]; then
            reservado_count=$((reservado_count + 1))
        fi
    done
    
    if [ $reservado_count -gt 1 ]; then
        echo -e "${RED}🚨 RACE CONDITION DETECTADO!${NC}"
        echo -e "${RED}   El asiento $seat_number está reservado en $reservado_count servidores${NC}"
        echo -e "${RED}   Esto demuestra el problema de concurrencia${NC}"
    elif [ $reservado_count -eq 1 ]; then
        echo -e "${YELLOW}⚠️  Solo un servidor tiene el asiento reservado${NC}"
        echo -e "${YELLOW}   Puede que el race condition no se haya manifestado esta vez${NC}"
        echo -e "${YELLOW}   Intenta ejecutar el script varias veces${NC}"
    else
        echo -e "${GREEN}✅ Ningún servidor tiene el asiento reservado${NC}"
        echo -e "${YELLOW}   Todas las peticiones fallaron (posible, pero inusual)${NC}"
    fi
    
    echo ""
}

# Función para ejecutar múltiples pruebas
run_multiple_tests() {
    local iterations=$1
    local seat_number=$2
    
    echo -e "${YELLOW}🔄 Ejecutando $iterations pruebas para el asiento $seat_number${NC}"
    echo ""
    
    race_conditions_detected=0
    
    for i in $(seq 1 $iterations); do
        echo -e "${BLUE}--- Prueba $i/$iterations ---${NC}"
        
        # Resetear servidores
        reset_servers
        sleep 1
        
        # Ejecutar prueba
        demonstrate_race_condition $seat_number
        
        # Verificar si hubo race condition
        reservado_count=0
        for j in {1..3}; do
            url="http://localhost:808$j/asiento/$seat_number"
            response=$(curl -s "$url")
            disponible=$(echo "$response" | grep -o '"disponible":[^,]*' | cut -d':' -f2)
            
            if [ "$disponible" = "false" ]; then
                reservado_count=$((reservado_count + 1))
            fi
        done
        
        if [ $reservado_count -gt 1 ]; then
            race_conditions_detected=$((race_conditions_detected + 1))
        fi
        
        echo ""
        sleep 2
    done
    
    echo -e "${YELLOW}📊 RESUMEN DE PRUEBAS:${NC}"
    echo -e "   Total de pruebas: $iterations"
    echo -e "   Race conditions detectados: $race_conditions_detected"
    echo -e "   Porcentaje de éxito: $(( (race_conditions_detected * 100) / iterations ))%"
    echo ""
}

# Menú principal
show_menu() {
    echo -e "${BLUE}🎯 OPCIONES DISPONIBLES:${NC}"
    echo "1. Verificar estado de servidores"
    echo "2. Resetear todos los servidores"
    echo "3. Demostrar race condition (asiento específico)"
    echo "4. Ejecutar múltiples pruebas"
    echo "5. Mostrar estado de todos los asientos"
    echo "6. Salir"
    echo ""
}

# Función para mostrar todos los asientos
show_all_seats() {
    echo -e "${BLUE}📊 Estado de todos los asientos:${NC}"
    
    for i in {1..3}; do
        echo -e "${YELLOW}Servidor $i (puerto 808$i):${NC}"
        url="http://localhost:808$i/estado"
        response=$(curl -s "$url")
        
        if [ $? -eq 0 ]; then
            disponibles=$(echo "$response" | grep -o '"disponibles":[^,]*' | cut -d':' -f2)
            reservados=$(echo "$response" | grep -o '"reservados":[^,]*' | cut -d':' -f2)
            total=$(echo "$response" | grep -o '"total_asientos":[^,]*' | cut -d':' -f2)
            
            echo "  Total: $total, Disponibles: $disponibles, Reservados: $reservados"
        else
            echo -e "  ${RED}ERROR al conectar${NC}"
        fi
    done
    echo ""
}

# Script principal
main() {
    echo -e "${GREEN}🎬 SISTEMA DE DEMOSTRACIÓN DE RACE CONDITIONS${NC}"
    echo -e "${GREEN}=============================================${NC}"
    echo ""
    echo "Este script demuestra race conditions en un sistema distribuido"
    echo "donde múltiples servidores manejan el mismo estado sin sincronización."
    echo ""
    
    # Verificar que curl esté disponible
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}❌ Error: curl no está instalado${NC}"
        echo "Por favor instala curl para ejecutar este script"
        exit 1
    fi
    
    # Verificar servidores inicialmente
    check_servers
    
    # Menú interactivo
    while true; do
        show_menu
        read -p "Selecciona una opción (1-6): " choice
        
        case $choice in
            1)
                check_servers
                ;;
            2)
                reset_servers
                ;;
            3)
                read -p "Ingresa el número de asiento (1-50): " seat
                if [[ $seat =~ ^[0-9]+$ ]] && [ $seat -ge 1 ] && [ $seat -le 50 ]; then
                    demonstrate_race_condition $seat
                else
                    echo -e "${RED}❌ Número de asiento inválido${NC}"
                fi
                ;;
            4)
                read -p "Número de pruebas a ejecutar: " iterations
                read -p "Número de asiento (1-50): " seat
                if [[ $iterations =~ ^[0-9]+$ ]] && [[ $seat =~ ^[0-9]+$ ]] && [ $seat -ge 1 ] && [ $seat -le 50 ]; then
                    run_multiple_tests $iterations $seat
                else
                    echo -e "${RED}❌ Valores inválidos${NC}"
                fi
                ;;
            5)
                show_all_seats
                ;;
            6)
                echo -e "${GREEN}👋 ¡Hasta luego!${NC}"
                exit 0
                ;;
            *)
                echo -e "${RED}❌ Opción inválida${NC}"
                ;;
        esac
        
        echo ""
        read -p "Presiona Enter para continuar..."
        echo ""
    done
}

# Ejecutar script principal
main