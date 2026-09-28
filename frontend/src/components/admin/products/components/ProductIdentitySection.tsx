import AdminCard from "@/components/admin/ui/components/AdminCard";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminInput from "@/components/admin/ui/components/AdminInput";
import AdminSectionHeader from "@/components/admin/ui/components/AdminSectionHeader";
import AdminTextarea from "@/components/admin/ui/components/AdminTextarea";

import ProductCategoryPicker from "./ProductCategoryPicker";

import styles from "../css/ProductIdentitySection.module.css";

export type ProductIdentityValue = {
  name: string;
  categoryId: string;
  brand: string;
  shortDescription: string;
  description: string;
};

type ProductIdentitySectionProps = {
  value: ProductIdentityValue;

  onChange: (
    next: ProductIdentityValue,
  ) => void;

  disabled?: boolean;
};

export default function ProductIdentitySection({
  value,
  onChange,
  disabled = false,
}: ProductIdentitySectionProps) {
  return (
    <AdminCard
      className={styles.card}
    >
      <AdminSectionHeader
        eyebrow="Catalog"
        title="Product details"
        description="Basic information customers and staff will see."
      />

      <div
        className={
          styles.fields
        }
      >
        <AdminField
          label="Product name"
          required
          className={
            styles.full
          }
        >
          <AdminInput
            value={value.name}
            disabled={disabled}
            placeholder="Premium Linen Panjabi"
            required
            onChange={(
              event,
            ) =>
              onChange({
                ...value,

                name:
                  event.target
                    .value,
              })
            }
          />
        </AdminField>

        <ProductCategoryPicker
          value={
            value.categoryId
          }
          disabled={disabled}
          onChange={(
            categoryId,
          ) =>
            onChange({
              ...value,
              categoryId,
            })
          }
        />

        <AdminField
          label="Brand"
        >
          <AdminInput
            value={value.brand}
            disabled={disabled}
            placeholder="Ene dei"
            onChange={(
              event,
            ) =>
              onChange({
                ...value,

                brand:
                  event.target
                    .value,
              })
            }
          />
        </AdminField>

        <AdminField
          label="Short description"
        >
          <AdminInput
            value={
              value.shortDescription
            }
            disabled={disabled}
            placeholder="Short storefront summary"
            onChange={(
              event,
            ) =>
              onChange({
                ...value,

                shortDescription:
                  event.target
                    .value,
              })
            }
          />
        </AdminField>

        <AdminField
          label="Description"
          className={
            styles.full
          }
        >
          <AdminTextarea
            value={
              value.description
            }
            disabled={disabled}
            placeholder="Describe the product, material, fit and anything buyers should know."
            onChange={(
              event,
            ) =>
              onChange({
                ...value,

                description:
                  event.target
                    .value,
              })
            }
          />
        </AdminField>
      </div>
    </AdminCard>
  );
}