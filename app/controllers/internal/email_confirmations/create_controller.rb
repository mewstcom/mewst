# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::CreateController < Internal::ApplicationController
  def call
    form = Internal::EmailConfirmationForm.new(email: params[:email], locale: I18n.locale)

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Internal::ResponseErrorSerializer.new(resources),
        status: :unprocessable_entity
      )
    end

    result = CreateEmailConfirmationUseCase.new.call(
      email: form.email.not_nil!,
      locale: form.locale.not_nil!
    )

    resource = Internal::EmailConfirmationResource.new(email_confirmation: result.email_confirmation)
    render(
      json: Internal::EmailConfirmationSerializer.new(resource),
      status: :created
    )
  end
end
