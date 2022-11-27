# typed: strict
# frozen_string_literal: true

class ConfirmEmailService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
  end

  sig { params(form: EmailConfirmationForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    account = Account.find_by(email: @form.email)

    if (@form.on_sign_up? && account) || (@form.on_sign_in? && !account) || (@form.on_update_email? && !account)
      return Result.new
    end

    email_confirmation = EmailConfirmation.create!(email_confirmation_attributes(account:))

    if @form.on_sign_up?
      EmailConfirmationMailer.sign_up_confirmation(T.must(email_confirmation.id), I18n.locale.to_s).deliver_later
    elsif @form.on_sign_in?
      EmailConfirmationMailer.sign_in_confirmation(T.must(email_confirmation.id), I18n.locale.to_s).deliver_later
    end

    Result.new
  end

  sig { params(account: T.nilable(Account)).returns(T::Hash[Symbol, T.untyped]) }
  private def email_confirmation_attributes(account:)
    {
      account:,
      email: @form.email,
      event: @form.event,
      token: SecureRandom.uuid,
      back: @form.back,
      expires_at: Time.current + EmailConfirmation::EXPIRES_IN
    }
  end
end
