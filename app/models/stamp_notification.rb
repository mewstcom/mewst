# typed: strict
# frozen_string_literal: true

class StampNotification < ApplicationRecord
  belongs_to :notification
  belongs_to :stamp

  delegate :post, :profile, to: :stamp
end
