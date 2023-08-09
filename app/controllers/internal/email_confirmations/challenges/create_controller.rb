# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::Challenges::CreateController < Internal::ApplicationController
  def call
    form = Internal::EmailConfirmationChallengeForm.new(
      email_confirmation_id: params[:email_confirmation_id],
      confirmation_code: params[:confirmation_code]
    )

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Panko::Response.new(
          errors: Panko::ArraySerializer.new(resources, each_serializer: Internal::ResponseErrorSerializer)
        ),
        status: :unprocessable_entity
      )
    end

    input = ConfirmEmailService::Input.from_internal_form(form:)
    ConfirmEmailService.new.call(input:)

    head :no_content
  end
end
