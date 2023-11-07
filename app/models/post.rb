# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  extend Enumerize

  include Discard::Model

  MAXIMUM_CONTENT_LENGTH = 500

  belongs_to :profile
  has_many :stamps, dependent: :restrict_with_exception

  scope :kept, -> { undiscarded.joins(:profile).merge(Profile.kept) }

  validates :content, length: {maximum: MAXIMUM_CONTENT_LENGTH}, presence: true

  sig { returns(String) }
  def timeline_score
    published_at.strftime("%s%L")
  end
end
