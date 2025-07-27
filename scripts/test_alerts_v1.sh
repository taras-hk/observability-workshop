#!/bin/bash

random_delay() {
    min=$1
    max=$2
    echo $(( ( RANDOM % (max - min + 1) ) + min ))
}

BASE_URL="http://localhost:8082"
counter=0

basic_business_errors=(
    "payment_fail"
    "user_missing" 
    "plan_invalid"
    "system_error"
)

echo "🔥 V1: Generating basic business errors to trigger alerts..."

while true; do
    for i in {1..3}; do
        echo "V1: Sending basic business error $counter..."
        
        error_type=${basic_business_errors[$RANDOM % ${#basic_business_errors[@]}]}
        
        case $error_type in
            "payment_fail")
                curl -s -X POST "$BASE_URL/v1/subscriptions" \
                    -H "Content-Type: application/json" \
                    -d "{\"user_id\":\"user_$counter\", \"plan\":\"premium\", \"payment\":\"failed\"}" > /dev/null 2>&1
                echo "   Basic payment failure simulation"
                ;;
            "user_missing")
                curl -s -X POST "$BASE_URL/v1/subscriptions" \
                    -H "Content-Type: application/json" \
                    -d "{\"plan\":\"premium\"}" > /dev/null 2>&1
                echo "   Missing user ID error"
                ;;
            "plan_invalid")
                curl -s -X POST "$BASE_URL/v1/subscriptions" \
                    -H "Content-Type: application/json" \
                    -d "{\"user_id\":\"user_$counter\", \"plan\":\"invalid_$counter\"}" > /dev/null 2>&1
                echo "   Invalid plan error"
                ;;
            "system_error")
                curl -s -X POST "$BASE_URL/v1/subscriptions" \
                    -H "Content-Type: application/json" \
                    -d "{invalid_json_$counter" > /dev/null 2>&1
                echo "   System JSON parsing error"
                ;;
        esac

        curl -s "$BASE_URL/v1/subscriptions/non_existent_$counter" > /dev/null 2>&1

        counter=$((counter + 1))
    done

    delay=$(random_delay 1 3)
    echo "V1: Waiting ${delay} seconds..."
    sleep $delay
    
    if [ $counter -gt 15 ]; then
        echo "V1: Basic error generation complete"
        break
    fi
done
