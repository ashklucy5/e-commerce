import type {
  ButtonHTMLAttributes,
  ReactNode,
} from "react";

import styles from "../css/AdminButton.module.css";

type AdminButtonVariant =
  | "primary"
  | "secondary"
  | "danger"
  | "ghost";

type AdminButtonProps =
  ButtonHTMLAttributes<HTMLButtonElement> & {
    children: ReactNode;
    variant?: AdminButtonVariant;
    fullWidth?: boolean;
  };

export default function AdminButton({
  children,
  variant = "primary",
  fullWidth = false,
  className = "",
  ...props
}: AdminButtonProps) {
  return (
    <button
      {...props}
      className={[
        styles.button,
        styles[variant],
        fullWidth
          ? styles.fullWidth
          : "",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    >
      {children}
    </button>
  );
}