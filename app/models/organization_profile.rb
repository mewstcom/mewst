# typed: strict
# frozen_string_literal: true

class OrganizationProfile < ApplicationRecord
  belongs_to :organization
  belongs_to :profile
end
