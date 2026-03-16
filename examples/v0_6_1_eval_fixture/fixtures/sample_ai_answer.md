The failure comes from `calculateDiscountedTotal` in `src/cart.js`.

The current implementation subtracts the fixed coupon once per item entry:

- `subtotal - coupon.value * items.length`

That is wrong for this checkout model.

The domain note says a fixed coupon should reduce the final order total once, not once per item entry.

Expected fix:

- change the fixed-coupon branch to subtract `coupon.value` exactly once
- keep the percent coupon logic unchanged
- keep the readable cart summary behavior unchanged

Recommended follow-up:

- patch `src/cart.js`
- keep the existing test as the regression guard
- add one more test later if you want to cover a zero-coupon case
