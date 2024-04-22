# typed: strict
# frozen_string_literal: true

class LinkDataFetcherForm < ApplicationForm
  attribute :target_url, :string

  validates :target_url, presence: true, url: true

  sig { void }
  def add_fetch_error!
    errors.add(:target_url, :fetch_error)
  end
end
