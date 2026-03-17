from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, field_validator


class ProductReview(BaseModel):
    product_name: str | None = Field(None, description="Product name")
    reviewer_name: str | None = Field(None, description="Reviewer name")
    rating: int | None = Field(None, description="Rating from 1 to 5")
    sentiment: Literal["positive", "mixed", "negative"] | None = Field(
        None,
        description="Overall review sentiment",
    )
    summary: str | None = Field(None, description="Short review summary")
    delivery_feedback: str | None = Field(None, description="Shipping or delivery feedback")
    verified_purchase: bool | None = Field(None, description="Whether the reviewer appears to be a verified purchaser")

    @field_validator("rating")
    @classmethod
    def validate_rating(cls, value: int | None) -> int | None:
        if value is None:
            return None
        if not isinstance(value, int):
            raise TypeError("Rating must be an integer or null.")
        if value < 1 or value > 5:
            raise ValueError("Rating must be between 1 and 5.")
        return value


class ProductReviewList(BaseModel):
    reviews: list[ProductReview]
