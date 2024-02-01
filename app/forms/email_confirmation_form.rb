# typed: strict
# frozen_string_literal: true

class EmailConfirmationForm < ApplicationForm
  attribute :email, :string

  validates :email, email: true, presence: true
end
