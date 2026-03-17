from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field

from copilot.tools import define_tool


class ProductReview(BaseModel):
    product_name: str | None = Field(None, description="Product name")
    rating: int | None = Field(None, ge=1, le=5, description="Rating 1-5")
    reviewer: str | None = Field(None, description="Reviewer name")
    summary: str | None = Field(None, description="Review summary")
    sentiment: Literal["positive", "negative", "neutral"] | None = Field(
        None,
        description="positive, negative, or neutral",
    )


class ReviewList(BaseModel):
    reviews: list[ProductReview]


class SubmitReviewsParams(BaseModel):
    reviews: list[dict] = Field(description="Structured product review objects to validate")


@define_tool(description="Validate and submit parsed product reviews. Input must be a JSON object with a top-level reviews array.")
async def submit_reviews(params: SubmitReviewsParams) -> str:
    """Validates the structured review data."""
    try:
        review_list = ReviewList.model_validate(params.model_dump(mode="json"))
        return f"Successfully validated {len(review_list.reviews)} reviews"
    except Exception as exc:
        return f"Validation failed: {exc}"
