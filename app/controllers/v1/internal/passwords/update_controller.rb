# typed: true
# frozen_string_literal: true

class V1::Internal::Passwords::UpdateController < V1::Internal::ApplicationController
  include ControllerConcerns::InternalAuthenticatable
  include ControllerConcerns::V1::FormErrorable

  def call
    form = V1::PasswordResetForm.new(
      email: params[:email],
      password: params[:new_password]
    )

    if form.invalid?
      return response_form_errors(resource_class: V1::FormErrorResource, errors: form.errors)
    end

    UpdatePasswordUseCase.new.call(
      email: form.email.not_nil!,
      password: form.password.not_nil!
    )

    head :no_content
  end
end
