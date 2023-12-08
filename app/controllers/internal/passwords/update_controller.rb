# typed: true
# frozen_string_literal: true

class Internal::Passwords::UpdateController < Internal::ApplicationController
  include Latest::FormErrorable

  def call
    form = Internal::PasswordResetForm.new(
      email: params[:email],
      password: params[:new_password]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    UpdatePasswordUseCase.new.call(
      email: form.email.not_nil!,
      password: form.password.not_nil!
    )

    head :no_content
  end
end
