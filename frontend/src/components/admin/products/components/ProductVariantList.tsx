import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminCard from "@/components/admin/ui/components/AdminCard";
import AdminSectionHeader from "@/components/admin/ui/components/AdminSectionHeader";

import ProductVariantCard, {
  type ProductVariantValue,
} from "./ProductVariantCard";

import styles from "../css/ProductCreateForm.module.css";

type ProductVariantListProps = {
  variants: ProductVariantValue[];

  disabled?: boolean;

  onChange: (
    variants: ProductVariantValue[],
  ) => void;
};

export function createEmptyVariant(): ProductVariantValue {
  return {
    colorName: "",
    size: "",
    minimumOrderQuantity: 1,
    priceBdt: "",
    stock: 0,
  };
}

export default function ProductVariantList({
  variants,
  disabled = false,
  onChange,
}: ProductVariantListProps) {
  function changeVariant(
    index: number,
    next: ProductVariantValue,
  ) {
    onChange(
      variants.map(
        (
          variant,
          currentIndex,
        ) =>
          currentIndex === index
            ? next
            : variant,
      ),
    );
  }

  function removeVariant(
    index: number,
  ) {
    onChange(
      variants.filter(
        (
          _,
          currentIndex,
        ) =>
          currentIndex !== index,
      ),
    );
  }

  function addVariant() {
    onChange([
      ...variants,
      createEmptyVariant(),
    ]);
  }

  return (
    <AdminCard
      className={
        styles.sectionCard
      }
    >
      <div
        className={
          styles.sectionTop
        }
      >
        <AdminSectionHeader
          eyebrow="Variants"
          title="Pricing & stock"
          description="Add only the sellable versions of this product. Technical IDs and SKUs are handled automatically."
        />

        <AdminButton
          type="button"
          variant="secondary"
          disabled={disabled}
          onClick={addVariant}
        >
          + Add variant
        </AdminButton>
      </div>

      <div
        className={
          styles.variantList
        }
      >
        {variants.map(
          (
            variant,
            index,
          ) => (
            <ProductVariantCard
              key={index}
              index={index}
              value={variant}
              removable={
                variants.length >
                1
              }
              disabled={disabled}
              onChange={(
                next,
              ) =>
                changeVariant(
                  index,
                  next,
                )
              }
              onRemove={() =>
                removeVariant(
                  index,
                )
              }
            />
          ),
        )}
      </div>
    </AdminCard>
  );
}