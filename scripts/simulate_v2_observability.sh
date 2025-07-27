#!/bin/bash

create_subscription_v2() {
    local plan=$1
    local user_id="v2_user_$(date +%s%N | cut -b1-13)"
    local response=$(curl -s -X POST http://localhost:8082/v2/subscriptions \
        -H "Content-Type: application/json" \
        -d "{\"user_id\":\"$user_id\", \"plan\":\"$plan\"}")
    echo $(echo $response | jq -r .id 2>/dev/null || echo "")
}

declare -a subscription_ids

echo "Starting V2 traffic simulation..."
echo ""

counter=0
while true; do
    echo "🔄 V2 Simulation cycle $counter"
    
    echo "📝 Creating subscriptions with better context..."
    for plan in "basic" "premium"; do
        for i in {1..2}; do
            sub_id=$(create_subscription_v2 "$plan")
            if [[ "$sub_id" != "" && "$sub_id" != "null" ]]; then
                subscription_ids+=("$sub_id")
                echo "   ✓ Created V2 $plan subscription: $sub_id"
            fi
        done
    done
    
    echo "📖 Generating read traffic with better observability..."
    for i in {1..3}; do
        echo "   → V2 List all subscriptions"
        curl -s http://localhost:8082/v2/subscriptions > /dev/null 2>&1
        
        if [ ${#subscription_ids[@]} -gt 0 ]; then
            rand_index=$((RANDOM % ${#subscription_ids[@]}))
            id=${subscription_ids[$rand_index]}
            echo "   → V2 Get subscription: $id"
            curl -s http://localhost:8082/v2/subscriptions/$id > /dev/null 2>&1
        fi
    done
    
    echo "❌ Generating errors (notice improved observability)..."
    for i in {1..2}; do
        echo "   → V2 Invalid JSON request"
        curl -s -X POST http://localhost:8082/v2/subscriptions \
            -H "Content-Type: application/json" \
            -d "{invalid_json" > /dev/null 2>&1
            
        echo "   → V2 Non-existent subscription request"
        curl -s http://localhost:8082/v2/subscriptions/nonexistent_v2_$counter > /dev/null 2>&1
    done
    
    if [ ${#subscription_ids[@]} -gt 0 ]; then
        rand_index=$((RANDOM % ${#subscription_ids[@]}))
        id=${subscription_ids[$rand_index]}
        echo "✏️  V2 Update subscription: $id"
        curl -s -X PUT http://localhost:8082/v2/subscriptions/$id \
            -H "Content-Type: application/json" \
            -d "{\"user_id\":\"updated_v2_user\", \"plan\":\"premium\"}" > /dev/null 2>&1
    fi
    
    if [ ${#subscription_ids[@]} -gt 5 ] && [ $((RANDOM % 3)) -eq 0 ]; then
        rand_index=$((RANDOM % ${#subscription_ids[@]}))
        id=${subscription_ids[$rand_index]}
        echo "🗑️  V2 Delete subscription: $id"
        curl -s -X DELETE http://localhost:8082/v2/subscriptions/$id > /dev/null 2>&1
        subscription_ids=("${subscription_ids[@]/$id}")
    fi
    
    echo ""
    echo "📊 Current V2 subscriptions: ${#subscription_ids[@]}"
    echo "⏱️  Waiting 4 seconds before next cycle..."
    echo "=================================================="
    
    counter=$((counter + 1))
    sleep 4
    
    if [ ${#subscription_ids[@]} -gt 15 ]; then
        subscription_ids=("${subscription_ids[@]:7}")
    fi
done
