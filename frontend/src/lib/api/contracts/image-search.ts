import type {
  ProductCard,
} from "./catalog";

export type ImageSearchProduct =
  ProductCard & {
    match_type?: string;
    matched_sku?: string;
  };

export type ImageSearchMeta = {
  mode: string;

  query: string;

  page: number;
  limit: number;

  total: number;
  total_pages: number;

  primary_total: number;
  similar_total: number;

  no_match: boolean;

  showing_similar: boolean;
};

export type ImageSearchResponse = {
  data:
    ImageSearchProduct[];

  meta:
    ImageSearchMeta;
};