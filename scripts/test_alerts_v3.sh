#!/bin/bash

BASE_URL="http://localhost:8082"
TEST_DURATION=${TEST_DURATION:-300}
SLO_ERROR_THRESHOLD=5
SLO_LATENCY_THRESHOLD=1000

counter=0
success_count=0
error_count=0
start_time=$(date +%s)

# Business scenarios with plans
get_business_plan() {
    case $1 in
        "new_customer_onboarding") echo "premium" ;;
        "customer_upgrade") echo "premium" ;;
        "customer_downgrade") echo "basic" ;;
        "subscription_renewal") echo "basic" ;;
        "enterprise_migration") echo "premium" ;;
        "trial_conversion") echo "premium" ;;
        "bulk_provisioning") echo "premium" ;;
        "compliance_audit") echo "basic" ;;
        *) echo "basic" ;;
    esac
}

# Business alert patterns descriptions
get_business_alert_description() {
    case $1 in
        "revenue_critical_payment_failure") echo "Payment processing SLO breach - direct revenue impact" ;;
        "customer_acquisition_blocking") echo "Signup funnel blocked - CAC efficiency degraded" ;;
        "enterprise_service_degradation") echo "Enterprise SLA violation - churn risk high" ;;
        "compliance_violation_detected") echo "Regulatory compliance breach - legal risk" ;;
        "fraud_pattern_detected") echo "Fraudulent activity pattern - security incident" ;;
        "capacity_limit_approaching") echo "System capacity approaching limits - scaling needed" ;;
        "data_quality_degradation") echo "Data quality metrics failing - analytics impact" ;;
        "integration_partner_failure") echo "Third-party integration down - service disruption" ;;
        *) echo "Unknown business alert pattern" ;;
    esac
}

# Error patterns descriptions
get_error_pattern_description() {
    case $1 in
        "database_overload") echo "Database performance degraded - all services affected" ;;
        "payment_service_down") echo "Payment gateway unavailable - revenue stream blocked" ;;
        "validation_cascade_failure") echo "Validation service failing - customer experience degraded" ;;
        "rate_limiting_aggressive") echo "Rate limiting triggered - legitimate users affected" ;;
        "resource_exhaustion") echo "System resources exhausted - service capacity reached" ;;
        "network_partition") echo "Network connectivity issues - distributed system impact" ;;
        "cache_invalidation_storm") echo "Cache invalidation causing load spike" ;;
        "circuit_breaker_open") echo "Circuit breakers open - downstream service protection" ;;
        *) echo "Unknown error pattern" ;;
    esac
}

calculate_error_rate() {
    total=$((success_count + error_count))
    if [ $total -gt 0 ]; then
        echo $((error_count * 100 / total))
    else
        echo "0"
    fi
}

simulate_business_scenario() {
    local scenario=$1
    local plan=$(get_business_plan "$scenario")
    local user_id="user_${scenario}_${counter}"
    
    echo "V3: Business scenario '$scenario' - creating $plan subscription for $user_id"
    
    response=$(curl -s -w "%{http_code}:%{time_total}" -X POST "$BASE_URL/v3/subscriptions" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $user_id" \
        -H "X-Tenant-ID: tenant-$scenario" \
        -H "X-Request-ID: req-v3-$scenario-$counter" \
        -H "X-Session-ID: session-$scenario-$RANDOM" \
        -H "X-Business-Context: scenario=$scenario" \
        -d "{
            \"user_id\": \"$user_id\",
            \"plan\": \"$plan\"
        }")
    
    http_code=$(echo $response | cut -d':' -f1)
    time_total=$(echo $response | cut -d':' -f2 | cut -d'.' -f1)
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        success_count=$((success_count + 1))
        echo "   ✅ Success (${time_total}ms)"
    else
        error_count=$((error_count + 1))
        echo "   ❌ Error: HTTP $http_code (${time_total}ms)"
    fi
    
    if [ "${time_total:-0}" -gt $SLO_LATENCY_THRESHOLD ]; then
        echo "   🚨 LATENCY SLO BREACH: ${time_total}ms > ${SLO_LATENCY_THRESHOLD}ms"
    fi
}

simulate_business_alert_pattern() {
    local pattern=$1
    local description=$(get_business_alert_description "$pattern")
    
    echo "🚨 BUSINESS ALERT: $pattern"
    echo "   Impact: $description"
    
    case $pattern in
        "revenue_critical_payment_failure")
            for i in {1..3}; do
                response=$(curl -s -w "%{http_code}:%{time_total}" -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-Business-Alert: revenue_critical" \
                    -H "X-Customer-Tier: enterprise" \
                    -H "X-Revenue-Impact: high" \
                    -H "X-SLO-Category: payment_processing" \
                    -d "{\"user_id\": \"enterprise_customer_$counter\", \"plan\": \"premium\", \"payment_amount\": 9999}")
                error_count=$((error_count + 1))
                echo "   → Enterprise payment failure: \$9999 transaction blocked"
            done
            ;;
        "customer_acquisition_blocking")
            for signup_step in "validation" "verification" "activation"; do
                curl -s -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-Business-Alert: acquisition_blocking" \
                    -H "X-Funnel-Step: $signup_step" \
                    -H "X-Customer-Journey: signup" \
                    -H "X-CAC-Impact: true" \
                    -d "{\"user_id\": \"signup_blocked_$counter\", \"step\": \"$signup_step\"}" > /dev/null 2>&1
                error_count=$((error_count + 1))
                echo "   → Signup funnel blocked at $signup_step step"
            done
            ;;
        "enterprise_service_degradation")
            for i in {1..5}; do
                timeout 2 curl -s -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-Business-Alert: sla_violation" \
                    -H "X-Customer-Tier: enterprise" \
                    -H "X-SLA-Threshold: 500ms" \
                    -H "X-Churn-Risk: high" \
                    -d "{\"user_id\": \"enterprise_sla_$counter\", \"plan\": \"premium\"}" > /dev/null 2>&1 || echo "   → Enterprise SLA violation: >500ms response time"
                error_count=$((error_count + 1))
            done
            ;;
        "compliance_violation_detected")
            for violation in "gdpr_data_retention" "pci_dss_payment" "sox_audit_trail"; do
                curl -s -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-Business-Alert: compliance_violation" \
                    -H "X-Compliance-Type: $violation" \
                    -H "X-Legal-Risk: high" \
                    -H "X-Audit-Required: true" \
                    -d "{\"user_id\": \"compliance_test_$counter\", \"violation\": \"$violation\"}" > /dev/null 2>&1
                error_count=$((error_count + 1))
                echo "   → Compliance violation: $violation detected"
            done
            ;;
        "fraud_pattern_detected")
            for pattern in "velocity_anomaly" "geo_suspicious" "payment_pattern"; do
                curl -s -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-Business-Alert: fraud_detected" \
                    -H "X-Fraud-Pattern: $pattern" \
                    -H "X-Security-Incident: true" \
                    -H "X-Risk-Score: 95" \
                    -d "{\"user_id\": \"suspicious_$counter\", \"pattern\": \"$pattern\"}" > /dev/null 2>&1
                error_count=$((error_count + 1))
                echo "   → Fraud pattern detected: $pattern (risk score: 95%)"
            done
            ;;
        "capacity_limit_approaching")
            for resource in "cpu" "memory" "database_connections"; do
                curl -s -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-Business-Alert: capacity_limit" \
                    -H "X-Resource-Type: $resource" \
                    -H "X-Utilization: 90%" \
                    -H "X-Scaling-Required: true" \
                    -d "{\"user_id\": \"capacity_test_$counter\", \"resource\": \"$resource\"}" > /dev/null 2>&1
                error_count=$((error_count + 1))
                echo "   → Capacity alert: $resource at 90% utilization"
            done
            ;;
    esac
}

simulate_error_pattern() {
    local pattern=$1
    local description=$(get_error_pattern_description "$pattern")
    
    echo "V3: Simulating error pattern '$pattern' - $description"
    
    case $pattern in
        "database_overload")
            for i in {1..5}; do
                timeout 1 curl -s "$BASE_URL/v3/subscriptions" \
                    -H "X-User-ID: db-overload-user-$counter" \
                    -H "X-Request-ID: req-v3-db-overload-$counter-$i" \
                    -H "X-Load-Test: database_overload" > /dev/null || echo "   Timeout (simulated DB overload)"
                error_count=$((error_count + 1))
            done
            ;;
        "payment_service_down")
            curl -s -X POST "$BASE_URL/v3/subscriptions" \
                -H "Content-Type: application/json" \
                -H "X-User-ID: payment-fail-user-$counter" \
                -H "X-Request-ID: req-v3-payment-fail-$counter" \
                -H "X-Error-Simulation: payment_service_down" \
                -d "{\"user_id\": \"payment_fail_user_$counter\", \"plan\": \"premium\"}" > /dev/null 2>&1
            error_count=$((error_count + 1))
            ;;
        "validation_failures")
            for field in "user_id" "plan"; do
                curl -s -X POST "$BASE_URL/v3/subscriptions" \
                    -H "Content-Type: application/json" \
                    -H "X-User-ID: validation-test-$counter" \
                    -H "X-Request-ID: req-v3-validation-$counter-$field" \
                    -H "X-Validation-Test: missing_$field" \
                    -d "{\"user_id\": \"\", \"plan\": \"\"}" > /dev/null 2>&1
                error_count=$((error_count + 1))
            done
            ;;
        "rate_limiting")
            echo "   Sending burst requests to trigger rate limiting..."
            for i in {1..10}; do
                curl -s "$BASE_URL/v3/subscriptions" \
                    -H "X-User-ID: rate-limit-user-$counter" \
                    -H "X-Request-ID: req-v3-rate-limit-$counter-$i" \
                    -H "X-Load-Test: rate_limiting" > /dev/null &
            done
            wait
            error_count=$((error_count + 5))
            ;;
        "resource_exhaustion")
            large_payload=$(printf '{"user_id":"resource_test_%s","plan":"premium","metadata":"%s"}' $counter $(head -c 1000 /dev/zero | tr '\0' 'x'))
            curl -s -X POST "$BASE_URL/v3/subscriptions" \
                -H "Content-Type: application/json" \
                -H "X-User-ID: resource-test-$counter" \
                -H "X-Request-ID: req-v3-resource-$counter" \
                -H "X-Load-Test: resource_exhaustion" \
                -d "$large_payload" > /dev/null 2>&1
            error_count=$((error_count + 1))
            ;;
    esac
}

while true; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))
    
    if [ $elapsed -gt $TEST_DURATION ]; then
        echo "V3: Test duration completed"
        break
    fi
    
    error_rate=$(calculate_error_rate)
    echo "V3: Current metrics - Errors: $error_count, Success: $success_count, Error Rate: ${error_rate}%"
    
    if [ $elapsed -lt 60 ]; then
        phase="warmup"
        echo "V3: Phase 1 - Warmup (normal operations + minor alerts)"
        scenarios=("new_customer_onboarding" "subscription_renewal")
        error_frequency=10
        business_alert_frequency=15
    elif [ $elapsed -lt 180 ]; then
        phase="escalation" 
        echo "V3: Phase 2 - Escalation (increasing errors + business alerts)"
        scenarios=("customer_upgrade" "customer_downgrade" "enterprise_migration")
        error_frequency=5
        business_alert_frequency=8
    else
        phase="slo_breach"
        echo "V3: Phase 3 - SLO Breach Simulation (critical business alerts)"
        scenarios=("enterprise_migration" "bulk_provisioning")
        error_frequency=3
        business_alert_frequency=4
    fi
    
    for scenario in "${scenarios[@]}"; do
        simulate_business_scenario "$scenario"
        counter=$((counter + 1))
        
        if [ $((counter % business_alert_frequency)) -eq 0 ]; then
            business_patterns_array=(
                "revenue_critical_payment_failure"
                "customer_acquisition_blocking"
                "enterprise_service_degradation"
                "compliance_violation_detected"
                "fraud_pattern_detected"
                "capacity_limit_approaching"
                "data_quality_degradation"
                "integration_partner_failure"
            )
            random_business_pattern=${business_patterns_array[$RANDOM % ${#business_patterns_array[@]}]}
            simulate_business_alert_pattern "$random_business_pattern"
        fi
        
        if [ $((counter % error_frequency)) -eq 0 ]; then
            error_patterns_array=(
                "database_overload"
                "payment_service_down"
                "validation_cascade_failure"
                "rate_limiting_aggressive"
                "resource_exhaustion"
                "network_partition"
                "cache_invalidation_storm"
                "circuit_breaker_open"
            )
            random_pattern=${error_patterns_array[$RANDOM % ${#error_patterns_array[@]}]}
            simulate_error_pattern "$random_pattern"
        fi
        
        sleep 0.5
    done
    
    if [ "$error_rate" -gt $SLO_ERROR_THRESHOLD ]; then
        echo "🚨 SLO BREACH DETECTED: Error rate ${error_rate}% > ${SLO_ERROR_THRESHOLD}%"
        echo "   This should trigger alerting systems!"
    fi
    
    case $phase in
        "warmup") sleep 3 ;;
        "escalation") sleep 2 ;;
        "slo_breach") sleep 1 ;;
    esac
done

total_requests=$((success_count + error_count))
final_error_rate=$(calculate_error_rate)
test_duration_actual=$(($(date +%s) - start_time))

if [ "$final_error_rate" -gt $SLO_ERROR_THRESHOLD ]; then
    echo "🚨 SLO BREACH CONFIRMED: Error rate ${final_error_rate}% exceeds ${SLO_ERROR_THRESHOLD}% threshold"
    echo "   ✅ Alert systems should have been triggered"
else
    echo "✅ SLO MAINTAINED: Error rate ${final_error_rate}% within ${SLO_ERROR_THRESHOLD}% threshold"
fi
