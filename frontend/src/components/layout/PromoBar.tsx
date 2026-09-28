import { Icon } from "@/components/ui/Icon";

const items = [
  { icon: "secureCheckout" as const, text: "Secure checkout" },
  { icon: "delivery" as const, text: "Delivery options at checkout" },
  { icon: "support" as const, text: "Business support" },
];

export function PromoBar() {
  return (
    <div className="promo-bar">
      <div className="site-container promo-bar__inner">
        <div className="promo-bar__messages" aria-label="Store benefits">
          {items.map((item, index) => (
            <span key={item.text} className="promo-bar__message">
              {index > 0 && <span className="promo-bar__dot" aria-hidden="true" />}
              <Icon name={item.icon} size={13} />
              <span>{item.text}</span>
            </span>
          ))}
        </div>

        <span className="promo-bar__business">Built for wholesale buying</span>
      </div>
    </div>
  );
}
