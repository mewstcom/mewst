# typed: true
# frozen_string_literal: true

class Settings::Users::UpdateController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = UserForm.new(form_params)

    if @form.invalid?
      return render_as_invalid
    end

    result = UpdateUserUseCase.new(client: v1_public_client).call(form: @form, actor: current_actor.not_nil!)

    if result.errors
      @form.add_use_case_errors(result.errors.not_nil!)
      return render_as_invalid
    end

    flash[:notice] = t("messages.users.updated")
    redirect_to settings_user_path
  end

  sig { returns(ActionController::Parameters) }
  private def form_params
    T.cast(params.require(:user_form), ActionController::Parameters).permit(:locale, :time_zone)
  end

  private def render_as_invalid
    render("settings/users/show/call", status: :unprocessable_entity)
  end
end
