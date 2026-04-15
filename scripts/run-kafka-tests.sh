#!/bin/bash

# Скрипт для запуска performance тестов Kafka консьюмера
# Использование: ./scripts/run-kafka-tests.sh [test-name]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Параметры по умолчанию
TEST_NAME="${1:-all}"
TIMEOUT="${TIMEOUT:-10m}"
VERBOSE="${VERBOSE:-true}"

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker не найден. Установите Docker."
        exit 1
    fi
    
    if ! docker ps &> /dev/null; then
        print_error "Docker не запущен."
        exit 1
    fi
    
    print_success "Docker запущен"
}

check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go не найден. Установите Go 1.25+."
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go версия: $GO_VERSION"
}

run_tests() {
    local test_pattern=$1
    local test_description=$2
    
    print_header "Запуск: $test_description"
    
    cd "$PROJECT_ROOT"
    
    if [ "$VERBOSE" = true ]; then
        go test -v -tags=integration -run="$test_pattern" -timeout="$TIMEOUT" \
            ./pkg/adapters/kafka/consumer/dql/... \
            -count=1
    else
        go test -tags=integration -run="$test_pattern" -timeout="$TIMEOUT" \
            ./pkg/adapters/kafka/consumer/dql/... \
            -count=1
    fi
    
    if [ $? -eq 0 ]; then
        print_success "Тесты пройдены: $test_description"
    else
        print_error "Тесты не пройдены: $test_description"
        return 1
    fi
}

# Основная логика
main() {
    print_header "Kafka Consumer Performance Tests"
    
    check_docker
    check_go
    
    echo ""
    
    case "$TEST_NAME" in
        "all")
            run_tests "TestConsumer" "Все тесты производительности"
            ;;
        "performance")
            run_tests "TestConsumerPerformance" "Сравнение Single vs Batch"
            ;;
        "throughput")
            run_tests "TestConsumerThroughput" "Максимальная пропускная способность"
            ;;
        "timeout")
            run_tests "TestConsumerBatchTimeout" "Тест таймаута батча"
            ;;
        "bench")
            run_tests "Benchmark" "Бенчмарки"
            ;;
        *)
            print_warning "Неизвестный тест: $TEST_NAME"
            echo "Доступные тесты: all, performance, throughput, timeout, bench"
            exit 1
            ;;
    esac
    
    echo ""
    print_header "Результаты"
    print_success "Все тесты завершены"
}

# Помощь
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo "Использование: $0 [test-name]"
    echo ""
    echo "Доступные тесты:"
    echo "  all          - Все тесты (по умолчанию)"
    echo "  performance  - Сравнение Single vs Batch режимов"
    echo "  throughput   - Измерение максимальной пропускной способности"
    echo "  timeout      - Тест таймаута батча"
    echo "  bench        - Бенчмарки"
    echo ""
    echo "Переменные окружения:"
    echo "  TIMEOUT      - Таймаут тестов (по умолчанию: 10m)"
    echo "  VERBOSE      - Подробный вывод (true/false)"
    echo ""
    echo "Примеры:"
    echo "  $0 performance"
    echo "  VERBOSE=false $0 throughput"
    echo "  TIMEOUT=15m $0 all"
    exit 0
fi

main
