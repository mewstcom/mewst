# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::CreateController < Internal::ApplicationController
  def call
    form = Forms::EmailConfirmation.new(email: params[:email], locale: I18n.locale)

    if form.invalid?
      return render(
        json: Internal::Resources::ActiveModelErrors.new(form.errors),
        status: :unprocessable_entity
      )
    end

    result = Services::SendEmailConfirmationCode.new(form:).call

    render(
      json: Internal::Resources::EmailConfirmation.new(result.email_confirmation),
      status: :created
    )
  end
end
