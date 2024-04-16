# typed: strict
# frozen_string_literal: true

class LinkForm < ApplicationForm
  attribute :canonical_url, :string
  attribute :domain, :string
  attribute :title, :string
  attribute :image_url, :string

  validates :canonical_url, presence: true, url: true
  validates :domain, presence: true
  validates :title, presence: true
  validates :image_url, url: {allow_blank: true}
end
