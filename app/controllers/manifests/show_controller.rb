# typed: true
# frozen_string_literal: true

class Manifests::ShowController < ApplicationController
  sig { returns(T.untyped) }
  def call
    respond_to do |format|
      format.json { render(layout: false) }
      format.any  { render(plain: "Not Found", status: :not_found) }
    end
  end
end
