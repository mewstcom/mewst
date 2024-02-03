# typed: true
# frozen_string_literal: true

class Stamps::CreateController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = StampForm.new(post_id: params[:post_id])

    if @form.invalid?
      return render(content_type: "text/vnd.turbo-stream.html", status: :unprocessable_entity, layout: false)
    end

    result = CreateStampUseCase.new(client: v1_public_client).call(form: @form)
    @post = result.stamp.not_nil!.post

    render(content_type: "text/vnd.turbo-stream.html", layout: false)
  end
end
