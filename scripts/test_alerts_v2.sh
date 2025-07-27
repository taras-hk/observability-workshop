#!/bin/bash

random_delay() {
    min=$1
    max=$2
    echo $(( ( RANDOM % (max - min + 1) ) + min ))
}

BASE_URL="http://localhost:8082"
counter=0

business_error_types=(
    "payment_processing_failure"
    "user_validation_error"
    "plan_quota_exceeded"
    "service_timeout"
    "rate_limit_exceeded"
)

echo "⚡ V2: Generating contextual business errors with patterns..."

for phase in "low" "medium" "high"; do
    case $phase in
        "low")
            error_rate=2
            delay_range_min=3
            delay_range_max=5
            duration=30
            ;;
        "medium")
            error_rate=5
            delay_range_min=2
            delay_range_max=4
            duration=45
            ;;
        "high")
            error_rate=8
            delay_range_min=1
            delay_range_max=2
            duration=60
            ;;
    esac
    
    echo "V2: Phase $phase - Error rate: $error_rate, Duration: ${duration}s"
    phase_start=$(date +%s)
    
    while [ $(($(date +%s) - phase_start)) -lt $duration ]; do
        for i in $(seq 1 $error_rate); do
            echo "V2: Sending contextual business error $counter..."
            
            error_type=${business_error_types[$RANDOM % ${#business_error_types[@]}]}
            
            case $error_type in
                "payment_processing_failure")
                    echo "V2: Payment processing failure $counter..."
                    curl -s -X POST "$BASE_URL/v2/subscriptions" \
                        -H "Content-Type: application/json" \
                        -H "X-User-ID: payment-user-$counter" \
                        -H "X-Request-ID: req-v2-payment-$counter" \
                        -H "X-Business-Context: revenue_impact" \
                        -d "{\"user_id\":\"payment_user_$counter\", \"plan\":\"premium\", \"payment_method\":\"invalid_card\"}" > /dev/null 2>&1
                    echo "   → Payment processing failed (revenue impact)"
                    ;;
                "user_validation_error")
                    echo "V2: User validation error $counter..."
                    curl -s -X POST "$BASE_URL/v2/subscriptions" \
                        -H "Content-Type: application/json" \
                        -H "X-Request-ID: req-v2-validation-$counter" \
                        -H "X-Business-Context: customer_experience" \
                        -d "{\"plan\":\"premium\"}" > /dev/null 2>&1
                    echo "   → User validation failed (customer experience impact)"
                    ;;
                "plan_quota_exceeded")
                    echo "V2: Plan quota exceeded $counter..."
                    curl -s -X POST "$BASE_URL/v2/subscriptions" \
                        -H "Content-Type: application/json" \
                        -H "X-User-ID: quota-user-$counter" \
                        -H "X-Request-ID: req-v2-quota-$counter" \
                        -H "X-Business-Context: capacity_planning" \
                        -d "{\"user_id\":\"quota_user_$counter\", \"plan\":\"enterprise_max\"}" > /dev/null 2>&1
                    echo "   → Plan quota exceeded (capacity planning needed)"
                    ;;
                "service_timeout")
                    echo "V2: Service timeout simulation $counter..."
                    timeout 2 curl -s -X POST "$BASE_URL/v2/subscriptions" \
                        -H "Content-Type: application/json" \
                        -H "X-User-ID: timeout-user-$counter" \
                        -H "X-Request-ID: req-v2-timeout-$counter" \
                        -H "X-Business-Context: performance_degradation" \
                        -d "{\"user_id\":\"timeout_user_$counter\", \"plan\":\"premium\"}" > /dev/null || echo "   → Timeout occurred (performance degradation)"
                    ;;
                "rate_limit_exceeded")
                    echo "V2: Rate limit exceeded simulation $counter..."
                    for j in {1..5}; do
                        curl -s -X POST "$BASE_URL/v2/subscriptions" \
                            -H "Content-Type: application/json" \
                            -H "X-User-ID: burst-user-${counter}-$j" \
                            -H "X-Request-ID: req-v2-burst-${counter}-$j" \
                            -H "X-Business-Context: abuse_prevention" \
                            -H "X-Rate-Limit-Test: burst" \
                            -d "{\"user_id\":\"burst_user_${counter}_$j\", \"plan\":\"basic\"}" > /dev/null &
                    done
                    wait
                    echo "   → Rate limit exceeded (burst requests)"
                    ;;
            esac
            
            counter=$((counter + 1))
        done
        
        delay=$(random_delay $delay_range_min $delay_range_max)
        echo "V2: Waiting ${delay} seconds (phase: $phase)..."
        sleep $delay
    done
    
    echo "V2: Phase $phase complete"
done
