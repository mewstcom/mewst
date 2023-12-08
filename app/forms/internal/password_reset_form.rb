# typed: strict
# frozen_string_literal: true

class Internal::PasswordResetForm < Internal::ApplicationForm
  include Form::PasswordValidatable

  attribute :email, :string
  attribute :password, :string

  validates :email, presence: true
end
