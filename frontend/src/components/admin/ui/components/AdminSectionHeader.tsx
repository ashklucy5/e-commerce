import styles from "../css/AdminSectionHeader.module.css";

type AdminSectionHeaderProps = {
  title: string;
  description?: string;
  eyebrow?: string;
};

export default function AdminSectionHeader({
  title,
  description,
  eyebrow,
}: AdminSectionHeaderProps) {
  return (
    <header className={styles.header}>
      {eyebrow ? (
        <p className={styles.eyebrow}>
          {eyebrow}
        </p>
      ) : null}

      <h2 className={styles.title}>
        {title}
      </h2>

      {description ? (
        <p className={styles.description}>
          {description}
        </p>
      ) : null}
    </header>
  );
}