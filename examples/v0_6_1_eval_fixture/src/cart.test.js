const test = require("node:test");
const assert = require("node:assert/strict");

const { calculateDiscountedTotal, summarizeCart } = require("./cart");

test("summarizeCart keeps cart names readable", () => {
  const items = [
    { name: "Notebook", price: 12, quantity: 2 },
    { name: "Pen", price: 3, quantity: 1 },
  ];

  assert.equal(summarizeCart(items), "Notebook x2, Pen x1");
});

test("fixed coupon is applied once to the order total", () => {
  const items = [
    { name: "Notebook", price: 12, quantity: 2 },
    { name: "Pen", price: 3, quantity: 1 },
  ];

  const total = calculateDiscountedTotal(items, { type: "fixed", value: 5 });

  assert.equal(total, 22);
});
