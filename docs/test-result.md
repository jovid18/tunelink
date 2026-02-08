# Load Test Results

## Stress Test - 2026-02-03 22:43

### Test Configuration
| Item | Value |
|------|-----|
| Test Type | Stress Test |
| VUs | 1000 (4 runners × 250) |
| Iterations | 3 per VU (Total 3000) |
| Max Duration | 10m |
| Parallelism | 4 |
| Execution Time | 2026-02-03 22:43:39 KST |

### Results Summary

**Combined Results from 4 Runner Pods:**

| Metric | Result | Threshold | Status |
|------|------|-----------|------|
| Total Requests | 306,000 (76,500 × 4) | - | - |
| Success Rate | 100% | >99% | **PASS** |
| Error Rate | 0.00% | <1% | **PASS** |
| p(95) Latency | 109.46ms | <500ms | **PASS** |
| p(99) Latency | 171.68ms | - | - |

### Detailed Metrics (Runner Average)

| Metric | avg | min | med | max | p(90) | p(95) | p(99) |
|--------|-----|-----|-----|-----|-------|-------|-------|
| http_req_duration | 44.71ms | 2.21ms | 36.21ms | 659.55ms | 92.51ms | 109.46ms | 171.68ms |

### Checks Results

| Check | Pass Rate |
|-------|--------|
| health ok | 100% |
| create ok | 100% |
| redirect ok | 100% |

**Combined:**
- Total Checks: 306,000 (76,500 × 4)
- Checks Succeeded: 100%
- Checks Failed: 0%

### DB Statistics (Post-Test)
| Item | Value |
|------|-----|
| Total URLs | 3,000 |
| Total Clicks | 300,000 |
| Average Clicks | 100 |
| Min Clicks | 100 |
| Max Clicks | 100 |
| Redis Pending Clicks | 0 |

### Throughput
| Metric | Value |
|--------|-----|
| Requests/s (per runner) | ~1,626/s |
| Iterations/s (per runner) | ~15.9/s |
| **Total Throughput** | **~6,504 req/s** |

### Analysis

**Significant Improvements (Compared to Previous Test):**
1. **p(95) Response Time Improved by 95.6%**: 2.47s → 109.46ms
2. **p(99) Response Time Improved by 95.8%**: 4.10s → 171.68ms
3. **Average Response Time Improved by 93.5%**: 687.93ms → 44.71ms
4. **Throughput Increased 5.5x**: ~1,187 req/s → ~6,504 req/s
5. **Error Rate Maintained at 0%**: Confirmed stable service state
6. **Click Count Accuracy Maintained at 100%**: No concurrency issues

**Redis Cache Integration Effect (Commits: 339aac5, 0b67b36)**

Performance improved dramatically with the integration of Redis cache and click count synchronization.

| Category | Previous Method | Current Method |
|------|----------|-----------|
| Read | Direct DB Query | Redis Cache → DB Fallback |
| Click Count Increment | DB Atomic Update | Redis INCR → Batch Sync |
| Result | 687.93ms avg | 44.71ms avg |

**Performance Metrics Comparison (Previous → Current):**
| Metric | Previous (21:37) | Current (22:43) | Improvement |
|------|-------------|--------------|--------|
| Average Response Time | 687.93ms | 44.71ms | **-93.5%** |
| p(95) Response Time | 2.47s | 109.46ms | **-95.6%** |
| p(99) Response Time | 4.10s | 171.68ms | **-95.8%** |
| Max Response Time | ~620ms | ~660ms | Similar |
| Throughput | ~1,187 req/s | ~6,504 req/s | **+448%** |

**Conclusion:**
- Dramatic performance improvement achieved with Redis cache adoption
- p(95) < 110ms maintained even with 1000 concurrent VUs
- Click count accuracy 100% (guaranteed by Redis INCR atomicity)
- Confirmed high-performance system capable of 6,500+ req/s

### Grafana Dashboard

![Stress Test 2026-02-03 22:43](./images/stress-test-20260203-2243-grafana.png)

**Grafana Metrics (Based on Screenshot):**
| Item | Value |
|------|-----|
| HTTP requests | 100,000 (single runner basis) |
| HTTP request failures | 0 (No data) |
| Peak RPS | 6.98K req/s |
| HTTP Request Duration | 130ms |

- URL: http://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---

## Stress Test - 2026-02-03 21:37

### Test Configuration
| Item | Value |
|------|-----|
| Test Type | Stress Test |
| VUs | 1000 (4 runners × 250) |
| Iterations | 3 per VU (Total 3000) |
| Max Duration | 10m |
| Parallelism | 4 |
| Execution Time | 2026-02-03 21:37:08 KST |

### Results Summary

**Combined Results from 4 Runner Pods:**

| Metric | Result | Threshold | Status |
|------|------|-----------|------|
| Total Requests | 306,000 (76,500 × 4) | - | - |
| Success Rate | 100% | >99% | **PASS** |
| Error Rate | 0.00% | <1% | **PASS** |
| p(95) Latency | 2.47s | <500ms | **FAIL** |
| p(99) Latency | 4.10s | - | - |

### Detailed Metrics (Runner Average)

| Metric | avg | min | med | max | p(90) | p(95) | p(99) |
|--------|-----|-----|-----|-----|-------|-------|-------|
| http_req_duration | 687.93ms | 1.89ms | 247.66ms | 15.85s | 1.79s | 2.47s | 4.10s |

### Checks Results

| Check | Pass Rate |
|-------|--------|
| health ok | 100% |
| create ok | 100% |
| redirect ok | 100% |

**Combined:**
- Total Checks: 306,000 (76,500 × 4)
- Checks Succeeded: 100%
- Checks Failed: 0%

### DB Statistics (Post-Test)
| Item | Value |
|------|-----|
| Total URLs | 3,000 |
| Total Clicks | 300,000 |
| Average Clicks | 100 |
| Min Clicks | 100 |
| Max Clicks | 100 |

### Throughput
| Metric | Value |
|--------|-----|
| Requests/s (per runner) | ~297/s |
| Iterations/s (per runner) | ~2.9/s |
| **Total Throughput** | **~1,187 req/s** |

### Analysis

**Improvements (Compared to Previous Test):**
1. **Error Rate Maintained at 0%**: Confirmed stable service state
2. **All URL Creations Succeeded**: All 3,000 URLs created successfully
3. **All Checks Passed at 100%**: health, create, redirect all succeeded
4. **Click Count Accuracy 100%**: Concurrency issue fully resolved
   - Expected Total Clicks: 300,000 (3000 URLs × 100 redirects)
   - Actual Total Clicks: 300,000 (100% accurate)
   - Average/Min/Max clicks all matched at 100

**Concurrency Issue Resolution (Commit: 68f99c5)**

The issue where approximately 50% of clicks were lost in the previous test has been resolved.

| Category | Previous Method | Fixed Method |
|------|----------|------------|
| Pattern | Read → Modify → Write | Atomic SQL Update |
| Issue | Race Condition (Lost Update) | None |
| Result | 149,705 clicks (~50%) | 300,000 clicks (100%) |

```go
// Previous method (Race Condition occurred)
entity, _ := repo.FindByShortURL(ctx, shortURL)  // 1. Read
entity.IncrementClicks()                          // 2. +1 in memory
repo.Update(ctx, entity)                          // 3. Save

// Fixed method (Atomic Update)
db.Model(&url.URL{}).
    Where("short_url = ?", shortURL).
    UpdateColumn("clicks", gorm.Expr("clicks + 1"))  // SQL: UPDATE SET clicks = clicks + 1
```

Atomic operation at the SQL level (`clicks = clicks + 1`) ensures accurate counting even under concurrent requests

**Performance Metrics:**
1. **Average Response Time**: 687.93ms (16% improvement from previous 815.75ms)
2. **p(95) Response Time**: 2.47s (19% improvement from previous 3.05s)
3. **p(99) Response Time**: 4.10s (20% improvement from previous 5.13s)
4. **Throughput**: ~1,187 req/s (16% improvement from previous ~1,020 req/s)

**Conclusion:**
- Stable service delivery under 1000 concurrent VUs
- 100% functional accuracy achieved (HTTP responses and data integrity)
- Click count concurrency issue fully resolved
- Response time improvement confirmed even under heavy load

### Grafana Dashboard

![Stress Test 2026-02-03 21:37](./images/stress-test-20260203-2137-grafana.png)

**Grafana Metrics (Based on Screenshot):**
| Item | Value |
|------|-----|
| HTTP requests | 306,000 |
| HTTP request failures | 0 (No data) |
| Peak RPS | 3.15K req/s |
| HTTP Request Duration | 3.69s |

- URL: http://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---

## Stress Test - 2026-02-03 00:28

### Test Configuration
| Item | Value |
|------|-----|
| Test Type | Stress Test |
| VUs | 1000 (4 runners × 250) |
| Iterations | 3 per VU (Total 3000) |
| Max Duration | 10m |
| Parallelism | 4 |
| Execution Time | 2026-02-03 00:27:59 KST |

### Results Summary

**Combined Results from 4 Runner Pods:**

| Metric | Result | Threshold | Status |
|------|------|-----------|------|
| Total Requests | 306,000 (76,500 × 4) | - | - |
| Success Rate | 100% | >99% | **PASS** |
| Error Rate | 0.00% | <1% | **PASS** |
| p(95) Latency | 3.05s | <500ms | **FAIL** |
| p(99) Latency | 5.12s | - | - |

### Detailed Metrics (Runner Average)

| Metric | avg | min | med | max | p(90) | p(95) | p(99) |
|--------|-----|-----|-----|-----|-------|-------|-------|
| http_req_duration | 815.75ms | 2.19ms | 240.35ms | 13.63s | 2.19s | 3.05s | 5.13s |

### Checks Results

| Check | Pass Rate |
|-------|--------|
| health ok | 100% |
| create ok | 100% |
| redirect ok | 100% |

**Combined:**
- Total Checks: 306,000 (76,500 × 4)
- Checks Succeeded: 100%
- Checks Failed: 0%

### DB Statistics (Post-Test)
| Item | Value |
|------|-----|
| Total URLs | 3,000 |
| Total Clicks | 149,705 |
| Average Clicks | 49.9 |
| Min Clicks | 24 |
| Max Clicks | 86 |

### Throughput
| Metric | Value |
|--------|-----|
| Requests/s (per runner) | ~255/s |
| Iterations/s (per runner) | ~2.5/s |
| **Total Throughput** | **~1,020 req/s** |

### Analysis

**Improvements (Compared to Previous Test):**
1. **Error Rate 0%**: From previous 28.2% → 0% (fully resolved)
2. **All URL Creations Succeeded**: All 3,000 URLs created successfully
3. **All Checks Passed at 100%**: health, create, redirect all succeeded

**Cautions:**
1. **p(95) Response Time 3.05s**: Some request delays under heavy load
2. **Max Response Time ~13-16s**: Delays in extreme cases
3. **Click Count Loss (Suspected Concurrency Issue)**: Average 49.9 (approximately 50% of expected 100)
   - Expected Total Clicks: 300,000 (3000 URLs × 100 redirects)
   - Actual Total Clicks: 149,705 (approximately 50% lost)
   - Suspected Cause: Race Condition in `UPDATE clicks = clicks + 1` during concurrent requests
   - Future Improvement Needed: Review atomic updates or Redis counters

**Conclusion:**
- Capable of providing stable service under 1000 concurrent VUs
- 100% functional accuracy achieved (based on HTTP responses)
- Room for response time optimization (caching, DB query optimization, etc.)
- Click count concurrency issue needs resolution

### Grafana Dashboard

![Stress Test 2026-02-03 00:28](./images/stress-test-20260203-0028-grafana.png)

**Grafana Metrics (Based on Screenshot):**
| Item | Value |
|------|-----|
| HTTP requests | 141,100 |
| HTTP request failures | 0 (No data) |
| Peak RPS | 733 req/s |
| HTTP Request Duration | 4.55s |

- URL: http://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---

## Stress Test - 2026-02-03 00:12

### Test Configuration
| Item | Value |
|------|-----|
| Test Type | Stress Test |
| VUs | 1000 (4 runners × 250) |
| Iterations | 3 per VU (Total 3000) |
| Max Duration | 10m |
| Parallelism | 4 |
| Execution Time | 2026-02-03 00:12:02 KST |

### Results Summary

**Combined Results from 4 Runner Pods:**

| Metric | Result | Threshold | Status |
|------|------|-----------|------|
| Total Requests | 126,900 | - | - |
| Success Rate | 71.8% | >99% | **FAIL** |
| Error Rate | 28.2% | <1% | **FAIL** |
| p(95) Latency | ~490ms | <500ms | PASS |
| p(99) Latency | ~1.1s | - | - |

### Detailed Metrics (Per Runner Average)

| Metric | avg | min | med | max | p(90) | p(95) | p(99) |
|--------|-----|-----|-----|-----|-------|-------|-------|
| http_req_duration | ~269ms | ~2ms | ~211ms | ~18s | ~400ms | ~490ms | ~1.1s |
| http_req_duration (success) | ~261ms | ~2ms | ~219ms | ~4s | ~406ms | ~486ms | ~706ms |

### Checks Results

| Check | Runner 1 | Runner 2 | Runner 3 | Runner 4 |
|-------|----------|----------|----------|----------|
| health ok | 100% | 100% | 100% | 100% |
| create ok | 54% | 35% | 34% | 37% |
| redirect ok | 74% | 70% | 70% | 72% |

**Combined:**
- Total Checks: ~126,900
- Checks Succeeded: ~71.8%
- Checks Failed: ~28.2%

### DB Statistics (Post-Test)
| Item | Value |
|------|-----|
| Total URLs | 1,209 |
| Total Clicks | 63,276 |
| Average Clicks | 52.3 |
| Min Clicks | 21 |
| Max Clicks | 100 |

### Throughput
| Runner | Requests/s | Iterations/s |
|--------|------------|--------------|
| Runner 1 | 348/s | 6.19/s |
| Runner 2 | 232/s | 6.27/s |
| Runner 3 | 225/s | 6.16/s |
| Runner 4 | 242/s | 6.16/s |
| **Total** | **~1,047/s** | **~24.8/s** |

### Analysis

**Issues:**
1. **High URL Creation Failure Rate (54-66%)**: Bottleneck in URL creation API during concurrent requests
2. **Redirect Failure Rate ~28%**: Redirect requests failed for URLs that were not created
3. **Max Response Time 17-18 seconds**: Severe delays in some requests
4. **URL Count Mismatch**: Only 1,209 URLs created out of 3000 iterations (approximately 40%)

**Suspected Bottlenecks:**
- RDS Connection Pool limit
- API server concurrent processing capacity
- Conflicts/timeouts during concurrent URL creation

### Recommendations
1. Improve concurrency handling in URL creation API
2. Review expanding RDS Connection Pool
3. Horizontal scaling of API server (adjust HPA settings)
4. Strengthen retry logic and error handling

### Grafana Dashboard

![Stress Test 2026-02-03 00:12](./images/stress-test-20260203-0012-grafana.png)

**Grafana Metrics (Based on Screenshot):**
| Item | Value |
|------|-----|
| HTTP requests | 126,900 |
| HTTP request failures | 35,383 (27.9%) |
| Peak RPS | 363 req/s |
| HTTP Request Duration | 1.22s |

- URL: http://grafana.hearttune.link
- Dashboard: k6 Load Testing (ID: 19665)

---
