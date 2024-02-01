# typed: true
# frozen_string_literal: true

class Stamps::DestroyController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = StampForm.new(post_id: params[:post_id])

    if @form.invalid?
      return render(
        "stamps/create/call",
        content_type: "text/vnd.turbo-stream.html",
        layout: false,
        status: :unprocessable_entity
      )
    end

    result = DeleteStampUseCase.new(client: v1_public_client).call(form: @form)
    @post = result.stamp.not_nil!.post

    render("stamps/create/call", content_type: "text/vnd.turbo-stream.html", layout: false)
  end
end
