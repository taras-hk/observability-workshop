#!/bin/bash

create_subscription_v1() {
    local plan=$1
    local user_id="v1_user_$(date +%s%N | cut -b1-13)"
    local response=$(curl -s -X POST http://localhost:8082/v1/subscriptions \
        -H "Content-Type: application/json" \
        -d "{\"user_id\":\"$user_id\", \"plan\":\"$plan\"}")
    echo $(echo $response | jq -r .id 2>/dev/null || echo "")
}

declare -a subscription_ids

echo "Starting V1 traffic simulation..."
echo ""

counter=0
while true; do
    echo "🔄 V1 Simulation cycle $counter"
    
    echo "📝 Creating subscriptions..."
    for plan in "basic" "premium"; do
        for i in {1..2}; do
            sub_id=$(create_subscription_v1 "$plan")
            if [[ "$sub_id" != "" && "$sub_id" != "null" ]]; then
                subscription_ids+=("$sub_id")
                echo "   ✓ Created V1 $plan subscription: $sub_id"
            fi
        done
    done
    
    echo "📖 Generating read traffic..."
    for i in {1..3}; do
        echo "   → V1 List all subscriptions"
        curl -s http://localhost:8082/v1/subscriptions > /dev/null 2>&1
        
        if [ ${#subscription_ids[@]} -gt 0 ]; then
            rand_index=$((RANDOM % ${#subscription_ids[@]}))
            id=${subscription_ids[$rand_index]}
            echo "   → V1 Get subscription: $id"
            curl -s http://localhost:8082/v1/subscriptions/$id > /dev/null 2>&1
        fi
    done
    
    echo "❌ Generating errors (notice poor observability)..."
    for i in {1..2}; do
        echo "   → V1 Invalid JSON request"
        curl -s -X POST http://localhost:8082/v1/subscriptions \
            -H "Content-Type: application/json" \
            -d '{"invalid": "json"' > /dev/null 2>&1
            
        echo "   → V1 Non-existent subscription request"
        curl -s http://localhost:8082/v1/subscriptions/nonexistent_v1_$counter > /dev/null 2>&1
    done
    
    if [ ${#subscription_ids[@]} -gt 0 ]; then
        rand_index=$((RANDOM % ${#subscription_ids[@]}))
        id=${subscription_ids[$rand_index]}
        echo "✏️  V1 Update subscription: $id"
        curl -s -X PUT http://localhost:8082/v1/subscriptions/$id \
            -H "Content-Type: application/json" \
            -d "{\"user_id\":\"updated_v1_user\", \"plan\":\"premium\"}" > /dev/null 2>&1
    fi
    
    echo ""
    echo "📊 Current V1 subscriptions: ${#subscription_ids[@]}"
    echo "⏱️  Waiting 3 seconds before next cycle..."
    echo "=================================================="
    
    counter=$((counter + 1))
    sleep 3
    
    if [ ${#subscription_ids[@]} -gt 10 ]; then
        subscription_ids=("${subscription_ids[@]:5}")
    fi
done
