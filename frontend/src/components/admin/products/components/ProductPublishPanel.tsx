import AdminButton from "@/components/admin/ui/components/AdminButton";
import AdminCard from "@/components/admin/ui/components/AdminCard";
import AdminField from "@/components/admin/ui/components/AdminField";
import AdminSectionHeader from "@/components/admin/ui/components/AdminSectionHeader";
import AdminSelect from "@/components/admin/ui/components/AdminSelect";

import styles from "../css/ProductPublishPanel.module.css";

type ProductPublishPanelProps = {
  status: string;
  busy: boolean;
  progress: string;
  error: string;
  createdCode: string;

  onStatusChange: (
    status: string,
  ) => void;
};

export default function ProductPublishPanel({
  status,
  busy,
  progress,
  error,
  createdCode,
  onStatusChange,
}: ProductPublishPanelProps) {
  return (
    <AdminCard className={styles.card}>
      <AdminSectionHeader
        eyebrow="Control"
        title="Save product"
        description="The product code is assigned automatically from its category namespace."
      />

      <AdminField label="Initial status">
        <AdminSelect
          value={status}
          disabled={busy}
          onChange={(event) =>
            onStatusChange(
              event.target.value,
            )
          }
        >
          <option value="draft">
            Draft
          </option>

          <option value="active">
            Active
          </option>
        </AdminSelect>
      </AdminField>

      <div className={styles.codeBox}>
        <p className={styles.codeLabel}>
          Product code
        </p>

        <p className={styles.code}>
          {createdCode ||
            "Generated on save"}
        </p>
      </div>

      <AdminButton
        type="submit"
        fullWidth
        disabled={busy}
      >
        {busy
          ? "Saving…"
          : "Save product"}
      </AdminButton>

      {progress ? (
        <p className={styles.progress}>
          {progress}
        </p>
      ) : null}

      {error ? (
        <p
          className={styles.error}
          role="alert"
        >
          {error}
        </p>
      ) : null}

      <p className={styles.note}>
        Product creation and image
        storage are separate backend
        operations but are orchestrated
        here behind one Save action.
      </p>
    </AdminCard>
  );
}