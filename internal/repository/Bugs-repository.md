# Bug Report: repository

**Analysis Date:** 2025-12-30
**Directory Analyzed:** `/home/tranmh/work/gassigeher-saas/internal/repository`
**Files Analyzed:** 19 repository files
**Bugs Found:** 5 bugs

---

## Summary

This analysis focused on functional bugs in the repository layer, examining 19 repository files for error handling issues, missing `rows.Err()` checks, SQL injection risks, resource leaks, and race conditions. The codebase shows good overall quality with proper parameterized queries throughout and consistent error handling patterns. However, 5 functional bugs were identified:

- **Critical:** 1 bug (missing rows.Err() check)
- **High:** 2 bugs (missing rows.Err() checks)
- **Medium:** 2 bugs (GetAll pagination missing rows.Err(), color category query missing rows.Err())

Most repositories properly check `rows.Err()` after iteration, use parameterized queries to prevent SQL injection, and handle nullable values correctly. The identified bugs are primarily missing `rows.Err()` checks in specific query methods, which could silently ignore database iteration errors.

**Positive findings:**
- All queries use parameterized statements (no SQL injection vulnerabilities)
- Consistent error wrapping with `fmt.Errorf`
- Proper handling of `sql.ErrNoRows` vs other errors
- Good use of `defer rows.Close()` pattern
- Race condition fixes already implemented (dog limit, promo codes, referral codes with optimistic locking)
- Safe int64 to int conversions with bounds checking

---

## Bugs

## Bug #1: Missing rows.Err() Check in SubscriptionRepository.GetAllPlans

**Description:**
The `GetAllPlans` method in `subscription_repository.go` fetches all active pricing plans but does not check for errors that may occur during row iteration. If an error occurs while scanning rows (e.g., network interruption, type mismatch, or database corruption), the method will silently return partial results without any indication that an error occurred. This could lead to incomplete data being displayed to users, potentially showing fewer subscription plans than actually exist in the database.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/subscription_repository.go`
- Function: `GetAllPlans`
- Lines: 22-56

**Steps to Reproduce:**
1. Have a database with multiple pricing plans
2. Simulate a database connection issue or data corruption during iteration
3. Call `GetAllPlans()`
4. Expected: Method should return an error indicating the iteration failed
5. Actual: Method returns successfully with partial results, no error reported

**Fix:**
Add `rows.Err()` check after the iteration loop to detect any errors that occurred during row scanning:

```diff
 	plans := []*models.PricingPlan{}
 	for rows.Next() {
 		plan := &models.PricingPlan{}
 		err := rows.Scan(
 			&plan.ID,
 			&plan.Name,
 			&plan.Slug,
 			&plan.MaxDogs,
 			&plan.PriceMonthly,
 			&plan.PriceYearly,
 			&plan.IsActive,
 			&plan.CreatedAt,
 		)
 		if err != nil {
 			return nil, fmt.Errorf("failed to scan pricing plan: %w", err)
 		}
 		plans = append(plans, plan)
 	}
+
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("error iterating pricing plans: %w", err)
+	}

 	return plans, nil
 }
```

This ensures any errors during iteration are properly detected and returned to the caller.

---

## Bug #2: Missing rows.Err() Check in PromoCodeRepository.GetAll

**Description:**
The `GetAll` method in `promo_code_repository.go` retrieves all promo codes with optional filtering but does not check `rows.Err()` after iteration. If an error occurs while iterating through rows (e.g., database connection lost, data type mismatch, or corrupted data), the function will return partial results without any error indication. This is particularly critical for promo codes as administrators may not be aware that some promo codes are missing from the list, leading to incorrect business decisions or overlooked active promotions.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/promo_code_repository.go`
- Function: `GetAll`
- Lines: 218-293

**Steps to Reproduce:**
1. Create multiple promo codes in the database
2. Simulate a database error during row iteration (e.g., network timeout)
3. Call `GetAll(false)`
4. Expected: Method should return an error indicating iteration failed
5. Actual: Method returns successfully with incomplete promo code list, no error reported

**Fix:**
Add `rows.Err()` check after the iteration completes:

```diff
 		codes = append(codes, code)
 	}
+
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("error iterating promo codes: %w", err)
+	}

 	return codes, nil
 }
```

This ensures any errors that occur during row iteration are properly detected and returned to the caller.

---

## Bug #3: Missing rows.Err() Check in MarketingRepository.ListCampaigns

**Description:**
The `ListCampaigns` method in `marketing_repository.go` retrieves all marketing campaigns but does not verify that row iteration completed successfully. While the method does `defer rows.Close()` and checks individual scan errors, it fails to check `rows.Err()` after the loop. This means if an error occurs during iteration (e.g., database connection interruption, data corruption, or timeout), the function will silently return partial campaign data without any indication that the result set is incomplete. For marketing campaigns, this could mean administrators are unaware of active campaigns, leading to incorrect campaign management decisions.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/marketing_repository.go`
- Function: `ListCampaigns`
- Lines: 24-42

**Steps to Reproduce:**
1. Create multiple marketing campaigns in the database
2. Simulate a database error during iteration (e.g., network interruption mid-query)
3. Call `ListCampaigns()`
4. Expected: Method should return an error indicating incomplete data
5. Actual: Method returns successfully with partial campaign list, no error reported

**Fix:**
Add `rows.Err()` check after the iteration loop:

```diff
 	var campaigns []*models.MarketingCampaign
 	for rows.Next() {
 		c := &models.MarketingCampaign{}
 		if err := rows.Scan(&c.ID, &c.Type, &c.Name, &c.Description, &c.Config, &c.IsActive, &c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt); err != nil {
 			return nil, err
 		}
 		campaigns = append(campaigns, c)
 	}
+
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("error iterating marketing campaigns: %w", err)
+	}
+
 	return campaigns, nil
 }
```

This ensures that any errors occurring during row iteration are properly detected and returned.

---

## Bug #4: Missing rows.Err() Check in MarketingRepository.ListReferralCodes

**Description:**
The `ListReferralCodes` method in `marketing_repository.go` retrieves all referral codes with a LEFT JOIN to tenant data, but fails to check `rows.Err()` after iteration. If an error occurs during row iteration (e.g., join operation failure, network timeout, or data corruption), the method will return partial results without any error indication. For referral codes, this is problematic because administrators managing referral programs may not be aware that some codes are missing from the list, potentially leading to customer service issues when users claim their codes aren't working when they actually exist but weren't displayed.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/marketing_repository.go`
- Function: `ListReferralCodes`
- Lines: 230-254

**Steps to Reproduce:**
1. Create multiple referral codes with associated tenants
2. Simulate a database error during JOIN operation or iteration
3. Call `ListReferralCodes()`
4. Expected: Method should return an error indicating incomplete data
5. Actual: Method returns successfully with incomplete referral code list

**Fix:**
Add `rows.Err()` check after the iteration loop:

```diff
 	var codes []*models.ReferralCode
 	for rows.Next() {
 		c := &models.ReferralCode{}
 		if err := rows.Scan(&c.ID, &c.Code, &c.ReferrerTenantID, &c.ReferrerEmail,
 			&c.DiscountMonthsReferrer, &c.DiscountMonthsReferee, &c.UsesCount, &c.MaxUses,
 			&c.IsActive, &c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt, &c.ReferrerTenantName); err != nil {
 			return nil, err
 		}
 		codes = append(codes, c)
 	}
+
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("error iterating referral codes: %w", err)
+	}
+
 	return codes, nil
 }
```

This ensures that errors during iteration are properly caught and reported.

---

## Bug #5: Missing rows.Err() Check in ColorCategoryRepository.FindAll

**Description:**
The `FindAll` method in `color_category_repository.go` retrieves all color categories for a tenant but does not check `rows.Err()` after iteration completes. While the method properly uses `defer rows.Close()` and checks individual scan errors, it fails to verify that the iteration completed successfully. If an error occurs during iteration (e.g., database timeout, network interruption, or data corruption), the function will silently return partial color category data. This is concerning for the color-based access control system, as users might be shown an incomplete list of available colors, potentially preventing legitimate users from requesting colors that actually exist in the system.

**Location:**
- File: `/home/tranmh/work/gassigeher-saas/internal/repository/color_category_repository.go`
- Function: `FindAll`
- Lines: 140-174

**Steps to Reproduce:**
1. Create multiple color categories for a tenant
2. Simulate a database error during row iteration (e.g., network timeout)
3. Call `FindAll(tenantID)`
4. Expected: Method should return an error indicating incomplete results
5. Actual: Method returns successfully with partial color category list

**Fix:**
Add `rows.Err()` check after the iteration loop:

```diff
 	colors := []*models.ColorCategory{}
 	for rows.Next() {
 		color := &models.ColorCategory{}
 		err := rows.Scan(
 			&color.ID,
 			&color.TenantID,
 			&color.Name,
 			&color.HexCode,
 			&color.PatternIcon,
 			&color.SortOrder,
 			&color.CreatedAt,
 			&color.UpdatedAt,
 		)
 		if err != nil {
 			return nil, fmt.Errorf("failed to scan color category: %w", err)
 		}
 		colors = append(colors, color)
 	}
+
+	if err := rows.Err(); err != nil {
+		return nil, fmt.Errorf("error iterating color categories: %w", err)
+	}

 	return colors, nil
 }
```

This ensures any errors during iteration are properly detected and returned to the caller.

---

## Statistics

- **Critical:** 1 bug
- **High:** 2 bugs
- **Medium:** 2 bugs
- **Low:** 0 bugs

---

## Recommendations

### High Priority
1. **Add rows.Err() checks to all methods with missing checks** - The 5 identified methods (`GetAllPlans`, `GetAll`, `ListCampaigns`, `ListReferralCodes`, `FindAll`) should all add `rows.Err()` checks immediately after their iteration loops. This is a simple fix that prevents silent data loss.

2. **Code review for consistency** - While most repository methods properly check `rows.Err()`, these 5 methods were missed. Conduct a final review to ensure no other methods are missing this check.

### Medium Priority
3. **Add integration tests for iteration errors** - Create tests that simulate database errors during row iteration to verify that these errors are properly caught and reported. This would have caught these bugs earlier.

4. **Linting rules** - Consider adding a custom linter rule or using `go vet` with additional checks to detect missing `rows.Err()` calls after `rows.Next()` loops.

### Best Practices Already Followed
- ✅ All queries use parameterized statements (no SQL injection vulnerabilities found)
- ✅ Consistent use of `defer rows.Close()` pattern
- ✅ Proper error wrapping with context using `fmt.Errorf`
- ✅ Good separation of `sql.ErrNoRows` handling vs other errors
- ✅ Race condition prevention in critical paths (dog limits, promo codes, referral codes)
- ✅ Safe int64 to int conversions with bounds checking (HIGH-8 fix)
- ✅ Tenant isolation properly enforced across all multi-tenant queries
- ✅ Nullable value handling with sql.Null* types
- ✅ Transaction handling with proper rollback in defer statements

### Code Quality Observations
The repository layer shows excellent overall code quality with:
- Consistent naming conventions and structure across all repositories
- Proper use of context keys for tenant isolation
- Good error messages that include context about what operation failed
- Comprehensive coverage of CRUD operations
- Good separation of concerns between simple queries and complex JOINs
- Proper handling of optional parameters with interface{} and sql.Null* types

The identified bugs are minor oversights in a few specific methods and do not indicate systemic issues with the codebase architecture or patterns.
