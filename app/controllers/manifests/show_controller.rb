# typed: true
# frozen_string_literal: true

class Manifests::ShowController < ApplicationController
  sig { returns(T.untyped) }
  def call
    render(layout: false)
  end
end
