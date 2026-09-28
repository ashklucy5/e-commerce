import type {
  TextareaHTMLAttributes,
} from "react";

import styles from "../css/AdminTextarea.module.css";

type AdminTextareaProps =
  TextareaHTMLAttributes<HTMLTextAreaElement> & {
    invalid?: boolean;
  };

export default function AdminTextarea({
  invalid = false,
  className = "",
  ...props
}: AdminTextareaProps) {
  return (
    <textarea
      {...props}
      className={[
        styles.textarea,
        invalid
          ? styles.invalid
          : "",
        className,
      ]
        .filter(Boolean)
        .join(" ")}
    />
  );
}