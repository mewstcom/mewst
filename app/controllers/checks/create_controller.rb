# typed: true
# frozen_string_literal: true

class Checks::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = SuggestedProfileForm.new(atname: params[:atname])

    if @form.invalid?
      return render(content_type: "text/vnd.turbo-stream.html", status: :unprocessable_entity, layout: false)
    end

    CheckSuggestedProfileUseCase.new(client: v1_public_client).call(form: @form)

    list_result = SuggestedProfileList.fetch(client: v1_public_client)

    if list_result.profiles.blank?
      redirect_to search_path
    else
      @atname = params[:atname]
      render(content_type: "text/vnd.turbo-stream.html", layout: false)
    end
  end
end
