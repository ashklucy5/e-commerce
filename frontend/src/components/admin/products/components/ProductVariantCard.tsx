import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminInput from "@/components/admin/ui/components/AdminInput";

import styles from "../css/ProductVariantCard.module.css";

export type ProductVariantValue = {
  colorName: string;
  size: string;
  minimumOrderQuantity: number;
  priceBdt: string;
  stock: number;
};

type ProductVariantCardProps = {
  index: number;
  value: ProductVariantValue;
  removable: boolean;
  disabled?: boolean;

  onChange: (
    next: ProductVariantValue,
  ) => void;

  onRemove: () => void;
};

export default function ProductVariantCard({
  index,
  value,
  removable,
  disabled = false,
  onChange,
  onRemove,
}: ProductVariantCardProps) {
  const variantName = [
    value.colorName.trim(),
    value.size.trim(),
  ]
    .filter(Boolean)
    .join(" / ");

  return (
    <article className={styles.variant}>
      <header className={styles.header}>
        <div>
          <p className={styles.eyebrow}>
            Variant {index + 1}
          </p>

          <h3 className={styles.title}>
            {variantName ||
              "Default variant"}
          </h3>

          <p className={styles.generated}>
            SKU will be generated
            automatically when saved.
          </p>
        </div>

        {removable ? (
          <AdminButton
            type="button"
            variant="ghost"
            disabled={disabled}
            onClick={onRemove}
          >
            Remove
          </AdminButton>
        ) : null}
      </header>

      <div className={styles.fields}>
        <AdminField
          label="Color"
          hint="Optional if this product has no color variants."
        >
          <AdminInput
            value={value.colorName}
            disabled={disabled}
            placeholder="Deep Blue"
            onChange={(event) =>
              onChange({
                ...value,
                colorName:
                  event.target.value,
              })
            }
          />
        </AdminField>

        <AdminField
          label="Size"
          hint="Optional if this product has no size variants."
        >
          <AdminInput
            value={value.size}
            disabled={disabled}
            placeholder="M"
            onChange={(event) =>
              onChange({
                ...value,
                size:
                  event.target.value,
              })
            }
          />
        </AdminField>

        <AdminField
          label="Minimum order quantity"
          hint="Smallest quantity a customer can order."
          required
        >
          <AdminInput
            type="number"
            min={1}
            step={1}
            value={
              value.minimumOrderQuantity
            }
            disabled={disabled}
            required
            onChange={(event) =>
              onChange({
                ...value,
                minimumOrderQuantity:
                  Number(
                    event.target.value,
                  ),
              })
            }
          />
        </AdminField>

        <AdminField
          label="Selling price"
          hint="Price per unit in Bangladeshi taka."
          required
        >
          <div className={styles.moneyField}>
            <span className={styles.currency}>
              ৳
            </span>

            <AdminInput
              type="text"
              inputMode="decimal"
              value={value.priceBdt}
              disabled={disabled}
              placeholder="1,250.00"
              required
              onChange={(event) =>
                onChange({
                  ...value,
                  priceBdt:
                    event.target.value,
                })
              }
            />
          </div>
        </AdminField>

        <AdminField
          label="Opening stock"
          hint="Available units currently in stock."
          required
        >
          <AdminInput
            type="number"
            min={0}
            step={1}
            value={value.stock}
            disabled={disabled}
            required
            onChange={(event) =>
              onChange({
                ...value,
                stock: Number(
                  event.target.value,
                ),
              })
            }
          />
        </AdminField>
      </div>
    </article>
  );
}