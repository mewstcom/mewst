# typed: strict
# frozen_string_literal: true

class Account < ApplicationRecord
  has_many :email_confirmations
  has_many :users

  validates :email, email: true, presence: true, uniqueness: true
end
