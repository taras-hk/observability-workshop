#!/bin/bash

create_subscription_v3() {
    local plan=$1
    local user_id="v3_user_$(date +%s%N | cut -b1-13)"
    local response=$(curl -s -X POST http://localhost:8082/v3/subscriptions \
        -H "Content-Type: application/json" \
        -d "{\"user_id\":\"$user_id\", \"plan\":\"$plan\"}")
    echo $(echo $response | jq -r .id 2>/dev/null || echo "")
}

declare -a subscription_ids

echo "Starting V3 traffic simulation..."
echo ""

counter=0
while true; do
    echo "🔄 V3 Simulation cycle $counter"
    
    echo "📝 Creating subscriptions with rich observability..."
    for plan in "basic" "premium"; do
        for i in {1..3}; do
            sub_id=$(create_subscription_v3 "$plan")
            if [[ "$sub_id" != "" && "$sub_id" != "null" ]]; then
                subscription_ids+=("$sub_id")
                echo "   ✓ Created V3 $plan subscription: $sub_id"
            fi
        done
    done
    
    echo "📖 Generating read traffic with full observability..."
    for i in {1..4}; do
        echo "   → V3 List all subscriptions"
        curl -s http://localhost:8082/v3/subscriptions > /dev/null 2>&1
        
        if [ ${#subscription_ids[@]} -gt 0 ]; then
            rand_index=$((RANDOM % ${#subscription_ids[@]}))
            id=${subscription_ids[$rand_index]}
            echo "   → V3 Get subscription: $id"
            curl -s http://localhost:8082/v3/subscriptions/$id > /dev/null 2>&1
        fi
    done
    
    echo "❌ Generating errors (notice excellent observability)..."
    for i in {1..2}; do
        echo "   → V3 Invalid JSON request"
        curl -s -X POST http://localhost:8082/v3/subscriptions \
            -H "Content-Type: application/json" \
            -d "{invalid_json" > /dev/null 2>&1
            
        echo "   → V3 Non-existent subscription request"
        curl -s http://localhost:8082/v3/subscriptions/nonexistent_v3_$counter > /dev/null 2>&1
    done
    
    if [ ${#subscription_ids[@]} -gt 0 ]; then
        rand_index=$((RANDOM % ${#subscription_ids[@]}))
        id=${subscription_ids[$rand_index]}
        echo "✏️  V3 Update subscription: $id"
        curl -s -X PUT http://localhost:8082/v3/subscriptions/$id \
            -H "Content-Type: application/json" \
            -d "{\"user_id\":\"updated_v3_user\", \"plan\":\"premium\"}" > /dev/null 2>&1
    fi
    
    if [ ${#subscription_ids[@]} -gt 3 ]; then
        rand_index=$((RANDOM % ${#subscription_ids[@]}))
        id=${subscription_ids[$rand_index]}
        echo "🗑️  V3 Delete subscription: $id"
        curl -s -X DELETE http://localhost:8082/v3/subscriptions/$id > /dev/null 2>&1
        subscription_ids=("${subscription_ids[@]/$id}")
    fi
    
    echo "🏢 Business operations simulation..."
    
    for i in {1..2}; do
        echo "   → V3 Bulk list operations"
        curl -s http://localhost:8082/v3/subscriptions > /dev/null 2>&1
    done
    
    if [ $((counter % 3)) -eq 0 ]; then
        echo "   → V3 Premium plan focus cycle"
        for i in {1..2}; do
            sub_id=$(create_subscription_v3 "premium")
            if [[ "$sub_id" != "" && "$sub_id" != "null" ]]; then
                subscription_ids+=("$sub_id")
                echo "     ✓ Created V3 premium subscription: $sub_id"
            fi
        done
    fi
    
    echo ""
    echo "📊 Current V3 subscriptions: ${#subscription_ids[@]}"
    echo "💡 V3 provides rich business insights and correlation"
    echo "⏱️  Waiting 5 seconds before next cycle..."
    echo "=================================================="
    
    counter=$((counter + 1))
    sleep 5
    
    if [ ${#subscription_ids[@]} -gt 20 ]; then
        subscription_ids=("${subscription_ids[@]:10}")
    fi
done
