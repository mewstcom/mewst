# typed: strict
# frozen_string_literal: true

class AccountProfile < ApplicationRecord
  belongs_to :account
  belongs_to :profile

  validates :account, uniqueness: true
  validates :profile, uniqueness: true
end
