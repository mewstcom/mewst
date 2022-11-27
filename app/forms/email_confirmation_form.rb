# typed: strict
# frozen_string_literal: true

class EmailConfirmationForm < ApplicationForm
  EVENT_SIGN_UP = "sign_up"
  EVENT_SIGN_IN = "sign_in"
  EVENT_UPDATE_EMAIL = "update_email"

  attribute :back, :string
  attribute :email, :string
  attribute :event, :string

  validates :back, format: {with: %r{\A/}, allow_blank: true}
  validates :email, email: true, presence: true
  validates :event, inclusion: {in: [EVENT_SIGN_UP, EVENT_SIGN_IN, EVENT_UPDATE_EMAIL]}

  sig { returns(T::Boolean) }
  def on_sign_up?
    event == EVENT_SIGN_UP
  end

  sig { returns(T::Boolean) }
  def on_sign_in?
    event == EVENT_SIGN_IN
  end

  sig { returns(T::Boolean) }
  def on_update_email?
    event == EVENT_UPDATE_EMAIL
  end
end
