# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  CONTENT_MAXIMUM_LENGTH = 500

  belongs_to :profile

  validates :content, length: {maximum: CONTENT_MAXIMUM_LENGTH}, presence: true

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(String) }
  def timeline_item_score
    created_at!.to_datetime.strftime("%Q")
  end

  sig { returns(ActiveSupport::TimeWithZone) }
  def created_at!
    T.cast(created_at, ActiveSupport::TimeWithZone)
  end
end
