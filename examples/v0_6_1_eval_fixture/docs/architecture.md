# Checkout Notes

This repo models a tiny checkout utility.

Important domain assumptions:

- a percent coupon reduces the subtotal by a percentage
- a fixed coupon reduces the final order total once
- the order summary should stay readable for support and debugging

Preferred language:

- use `order total` instead of `grand total`
- use `fixed coupon` instead of `flat discount`
- preserve the names of checkout and coupon concepts across follow-up prompts
