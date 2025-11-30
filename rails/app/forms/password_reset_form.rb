# typed: strict
# frozen_string_literal: true

class PasswordResetForm < ApplicationForm
  include FormConcerns::PasswordValidatable

  attribute :password, :string
end
