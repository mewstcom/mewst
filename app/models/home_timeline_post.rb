# typed: strict
# frozen_string_literal: true

class HomeTimelinePost < ApplicationRecord
  belongs_to :post
  belongs_to :profile
end
