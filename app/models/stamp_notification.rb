# typed: strict
# frozen_string_literal: true

class StampNotification < ApplicationRecord
  belongs_to :notification
  belongs_to :post
  belongs_to :profile
end
