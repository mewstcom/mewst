# typed: strict
# frozen_string_literal: true

class Actor < ApplicationRecord
  belongs_to :user
  belongs_to :profile

  delegate :following?, :follows, :home_timeline, :me?, :posts, :stamps, to: :profile
end
