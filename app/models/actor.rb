# typed: strict
# frozen_string_literal: true

class Actor < ApplicationRecord
  belongs_to :user
  belongs_to :profile
  has_many :oauth_access_tokens, dependent: :restrict_with_exception, foreign_key: :resource_owner_id, inverse_of: :resource_owner

  delegate :atname, :following?, :follows, :home_timeline, :me?, :posts, :stamps, to: :profile
end
