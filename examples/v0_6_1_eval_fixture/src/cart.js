function summarizeCart(items) {
  return items.map((item) => `${item.name} x${item.quantity}`).join(", ");
}

function calculateDiscountedTotal(items, coupon) {
  const subtotal = items.reduce((sum, item) => sum + item.price * item.quantity, 0);

  if (!coupon) {
    return subtotal;
  }

  if (coupon.type === "percent") {
    return subtotal - (subtotal * coupon.value) / 100;
  }

  if (coupon.type === "fixed") {
    return subtotal - coupon.value * items.length;
  }

  return subtotal;
}

module.exports = {
  summarizeCart,
  calculateDiscountedTotal,
};
