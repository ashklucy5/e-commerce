export type SearchSuggestionCategory = {
  id?: string;
  name: string;
  slug: string;
};

export type SearchSuggestionProduct = {
  id: string;
  product_code: string;
  name: string;
  slug: string;

  brand?: string;
  short_description?: string;

  is_featured: boolean;

  primary_image_url?: string;

  price_amount: number;
  currency: string;

  in_stock: boolean;

  matched_sku?: string;

  category: SearchSuggestionCategory;

  match_type: string;
};

export type SearchSuggestions = {
  query: string;

  products: SearchSuggestionProduct[];

  categories: SearchSuggestionCategory[];

  brands: string[];
};

export type SearchSuggestionsResponse = {
  data: SearchSuggestions;
};